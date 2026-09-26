package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/businesses"
	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/platform/security"
	"github.com/adera-platform/backend/internal/reviews"
	"github.com/adera-platform/backend/internal/targets"
	"github.com/adera-platform/backend/internal/users"
)

// Development seed data. Refused in production. Credentials below work ONLY
// in development and are documented in the README:
//
//	admin:     SEED_ADMIN_EMAIL (default admin@adera.local) / SEED_ADMIN_PASSWORD (default "admin12345!")
//	customers: abebe@example.com … / "password123"
//	owner:     owner@kategna.example.com / "password123"
func seed(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) error {
	if !cfg.IsDev() {
		return fmt.Errorf("seed refuses to run in production")
	}

	var seeded bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`,
		cfg.SeedAdminEmail).Scan(&seeded); err != nil {
		return fmt.Errorf("checking for existing seed: %w", err)
	}
	if seeded {
		slog.Info("seed data already present; nothing to do")
		return nil
	}

	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:   cfg.ArgonMemoryKiB,
		Iterations:  cfg.ArgonIterations,
		Parallelism: cfg.ArgonParallelism,
	})
	usersRepo := users.NewRepo(pool)
	targetsRepo := targets.NewRepo(pool)
	reviewsRepo := reviews.NewRepo(pool, cfg.StoragePublicBaseURL)
	bizRepo := businesses.NewRepo(pool, nil)

	newUser := func(name, email, password string) (uuid.UUID, error) {
		hash, err := hasher.Hash(password)
		if err != nil {
			return uuid.Nil, err
		}
		u := users.User{
			ID: uuid.New(), DisplayName: name, Email: email,
			PasswordHash: hash, Status: users.StatusActive, PreferredLanguage: "en",
		}
		if err := usersRepo.Create(ctx, u); err != nil {
			return uuid.Nil, fmt.Errorf("creating user %s: %w", email, err)
		}
		return u.ID, nil
	}

	adminPassword := cfg.SeedAdminPassword
	if adminPassword == "" {
		adminPassword = "admin12345!"
	}
	adminID, err := newUser("Adera Admin", cfg.SeedAdminEmail, adminPassword)
	if err != nil {
		return err
	}
	for _, role := range []string{"admin", "moderator"} {
		if err := usersRepo.GrantRole(ctx, adminID, role, adminID); err != nil {
			return err
		}
	}

	type customer struct {
		id   uuid.UUID
		name string
	}
	customerDefs := []struct{ name, email string }{
		{"Abebe Kebede", "abebe@example.com"},
		{"Tigist Alemu", "tigist@example.com"},
		{"Dawit Haile", "dawit@example.com"},
		{"Sara Tesfaye", "sara@example.com"},
		{"Yonas Girma", "yonas@example.com"},
	}
	customers := make([]customer, 0, len(customerDefs))
	for _, c := range customerDefs {
		id, err := newUser(c.name, c.email, "password123")
		if err != nil {
			return err
		}
		customers = append(customers, customer{id: id, name: c.name})
	}
	ownerID, err := newUser("Kategna Management", "owner@kategna.example.com", "password123")
	if err != nil {
		return err
	}

	kategnaBiz, err := bizRepo.Create(ctx, "Kategna Restaurant PLC",
		"Traditional Ethiopian restaurant group", adminID)
	if err != nil {
		return err
	}
	// Simulate an approved claim: membership + role + claimed status.
	if _, err := pool.Exec(ctx, `
		INSERT INTO business_members (business_id, user_id, role, granted_by) VALUES ($1, $2, 'owner', $3)`,
		kategnaBiz.ID, ownerID, adminID); err != nil {
		return fmt.Errorf("seeding membership: %w", err)
	}
	if err := usersRepo.GrantRole(ctx, ownerID, "business_owner", adminID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `UPDATE businesses SET verification_status = 'claimed' WHERE id = $1`, kategnaBiz.ID); err != nil {
		return fmt.Errorf("marking business claimed: %w", err)
	}

	// Fixed category IDs from migrations/0010_reference_data.sql.
	catElectronics := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	catBeauty := uuid.MustParse("11111111-1111-4111-8111-111111111102")
	catOnline := uuid.MustParse("11111111-1111-4111-8111-111111111103")
	catFood := uuid.MustParse("11111111-1111-4111-8111-111111111104")
	addis := uuid.MustParse("22222222-2222-4222-8222-222222222201")

	areaID := func(name string) (*uuid.UUID, error) {
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT id FROM areas WHERE city_id = $1 AND name = $2`, addis, name).Scan(&id); err != nil {
			return nil, fmt.Errorf("looking up area %s: %w", name, err)
		}
		return &id, nil
	}
	bole, err := areaID("Bole")
	if err != nil {
		return err
	}
	piassa, err := areaID("Piassa")
	if err != nil {
		return err
	}

	// at returns a coordinate pair for seeded targets. The positions are
	// approximate neighbourhood centroids for Piassa and Bole, good enough to
	// exercise /targets/nearby in development — not surveyed locations.
	at := func(lat, lng float64) (*float64, *float64) { return &lat, &lng }

	newTarget := func(in targets.CreateInput) (targets.Target, error) {
		in.CreatedBy = adminID
		in.AutoPublish = true
		in.CityID = &addis
		t, err := targetsRepo.Create(ctx, in)
		if err != nil {
			return targets.Target{}, fmt.Errorf("creating target %s: %w", in.Name, err)
		}
		return t, nil
	}

	shegerLat, shegerLng := at(9.0345, 38.748)
	sheger, err := newTarget(targets.CreateInput{
		TargetType: "repair_provider", CategoryID: catElectronics,
		Name: "Sheger Phone Repair", Description: "Phone and laptop repair near Piassa; screen and battery specialists.",
		AreaID: piassa, AddressText: "Piassa, behind Cinema Ethiopia", Aliases: []string{"ሸገር ስልክ ጥገና"},
		Latitude: shegerLat, Longitude: shegerLng,
	})
	if err != nil {
		return err
	}
	boleBeautyLat, boleBeautyLng := at(9.0104, 38.7869)
	boleBeauty, err := newTarget(targets.CreateInput{
		TargetType: "service", CategoryID: catBeauty,
		Name: "Bole Beauty Lounge", Description: "Hair, nails, and bridal packages around Bole Medhanialem.",
		AreaID: bole, AddressText: "Bole Medhanialem, next to Friendship Mall", Aliases: []string{"ቦሌ ውበት ሳሎን"},
		Latitude: boleBeautyLat, Longitude: boleBeautyLng,
	})
	if err != nil {
		return err
	}
	selam, err := newTarget(targets.CreateInput{
		TargetType: "online_seller", CategoryID: catOnline,
		Name: "Selam Online Store", Description: "Telegram shop for phone accessories and small electronics with city-wide delivery.",
		OnlineOnly: true, Aliases: []string{"ሰላም ኦንላይን"},
		SocialLinks: map[string]string{"telegram": "https://t.me/selamstore"},
	})
	if err != nil {
		return err
	}
	kategnaLat, kategnaLng := at(9.0113, 38.7897)
	kategna, err := newTarget(targets.CreateInput{
		TargetType: "restaurant", CategoryID: catFood, BusinessID: &kategnaBiz.ID,
		Name: "Kategna Restaurant Bole", Description: "Traditional Ethiopian dishes; famous for kitfo and shiro.",
		AreaID: bole, AddressText: "Bole, near Edna Mall", Aliases: []string{"ካተኛ", "Kategna Bole"},
		SocialLinks: map[string]string{"tiktok": "https://www.tiktok.com/@kategna.example"},
		Latitude:    kategnaLat, Longitude: kategnaLng,
	})
	if err != nil {
		return err
	}
	tomocaLat, tomocaLng := at(9.0352, 38.7492)
	tomoca, err := newTarget(targets.CreateInput{
		TargetType: "cafe", CategoryID: catFood,
		Name: "Tomoca Coffee Piassa", Description: "Historic Addis coffee house; strong macchiato, takeaway beans.",
		AreaID: piassa, AddressText: "Wavel Street, Piassa", Aliases: []string{"ቶሞካ ቡና", "Tomoca"},
		Latitude: tomocaLat, Longitude: tomocaLng,
	})
	if err != nil {
		return err
	}

	newReview := func(userIdx int, in reviews.Input) (reviews.Review, error) {
		rv, err := reviewsRepo.Create(ctx, customers[userIdx].id, in)
		if err != nil {
			return reviews.Review{}, fmt.Errorf("seeding review by %s: %w", customers[userIdx].name, err)
		}
		return rv, nil
	}
	b := func(v bool) *bool { return &v }
	i := func(v int) *int { return &v }
	f := func(v float64) *float64 { return &v }

	// Kategna: five reviews, all social discovery — exercises the Reality
	// Check percentages (n >= 5).
	kategnaReviews := []struct {
		user int
		in   reviews.Input
	}{
		{0, reviews.Input{TargetID: kategna.ID, OverallRating: 5, Title: "Better than the videos",
			Body: "የቲክቶክ ቪዲዮው ትክክል ነበር። ክትፎው በጣም ጣፋጭ ነው፣ አገልግሎቱም ፈጣን ነበር።", Language: "am",
			PricePaid: f(850), WouldRecommend: b(true), ReturnLikelihood: i(5),
			DiscoverySource: "tiktok", ExpectationMatch: "better",
			SocialMediaURL:  "https://www.tiktok.com/@foodie/video/1",
			CriterionScores: map[string]int{"taste": 5, "portion_size": 4, "price_fairness": 4, "service_speed": 5, "hygiene": 5, "atmosphere": 5, "social_media_accuracy": 5}}},
		{1, reviews.Input{TargetID: kategna.ID, OverallRating: 4, Title: "Solid, as advertised",
			Body: "Came after seeing it on TikTok. The kitfo matched the hype, portions are fair for the price.", Language: "en",
			PricePaid: f(700), WouldRecommend: b(true), ReturnLikelihood: i(4),
			DiscoverySource: "tiktok", ExpectationMatch: "as_expected",
			CriterionScores: map[string]int{"taste": 4, "portion_size": 4, "price_fairness": 4, "service_speed": 3, "hygiene": 4, "social_media_accuracy": 4}}},
		{2, reviews.Input{TargetID: kategna.ID, OverallRating: 4, Title: "Good food, slow on weekends",
			Body: "Discovered through Instagram reels. Food was as expected but we waited 40 minutes on a Saturday.", Language: "en",
			WouldRecommend: b(true), ReturnLikelihood: i(4),
			DiscoverySource: "instagram", ExpectationMatch: "as_expected",
			CriterionScores: map[string]int{"taste": 4, "service_speed": 2, "atmosphere": 4, "social_media_accuracy": 4}}},
		{3, reviews.Input{TargetID: kategna.ID, OverallRating: 3, Title: "Smaller portions than the video",
			Body: "ቪዲዮው ላይ የሚታየው መጠን ትልቅ ነው፤ በእውነቱ ትንሽ ነው። ጣዕሙ ግን ጥሩ ነው።", Language: "am",
			PricePaid: f(900), WouldRecommend: b(false), ReturnLikelihood: i(3),
			DiscoverySource: "tiktok", ExpectationMatch: "worse",
			CriterionScores: map[string]int{"taste": 4, "portion_size": 2, "price_fairness": 2, "social_media_accuracy": 2}}},
		{4, reviews.Input{TargetID: kategna.ID, OverallRating: 2, Title: "Nothing like the reels",
			Body: "The place in the videos looks spacious and calm; in reality it was crowded and the shiro was cold.", Language: "en",
			WouldRecommend: b(false), ReturnLikelihood: i(2),
			DiscoverySource: "instagram", ExpectationMatch: "very_different",
			CriterionScores: map[string]int{"taste": 2, "hygiene": 3, "atmosphere": 2, "social_media_accuracy": 1}}},
	}
	var firstKategnaReview reviews.Review
	for idx, kr := range kategnaReviews {
		rv, err := newReview(kr.user, kr.in)
		if err != nil {
			return err
		}
		if idx == 0 {
			firstKategnaReview = rv
		}
	}

	otherReviews := []struct {
		user int
		in   reviews.Input
	}{
		{0, reviews.Input{TargetID: sheger.ID, OverallRating: 5, Title: "Honest screen replacement",
			Body: "Replaced my Samsung screen with an original part and showed me the packaging first. Fair price.", Language: "en",
			PricePaid: f(4500), WouldRecommend: b(true),
			DiscoverySource: "friend",
			CriterionScores: map[string]int{"product_originality": 5, "repair_quality": 5, "seller_honesty": 5, "price_fairness": 4, "warranty": 4}}},
		{1, reviews.Input{TargetID: sheger.ID, OverallRating: 4, Title: "ጥሩ ጥገና",
			Body: "ባትሪ ቀይሮልኝ እስካሁን ችግር የለውም። ዋጋው ትንሽ ውድ ነው ግን ስራው ጥሩ ነው።", Language: "am",
			PricePaid: f(2200), WouldRecommend: b(true),
			DiscoverySource: "walk_in",
			CriterionScores: map[string]int{"repair_quality": 4, "price_fairness": 3, "repair_durability": 4}}},
		{2, reviews.Input{TargetID: boleBeauty.ID, OverallRating: 5, Title: "Kept the appointment time",
			Body: "Booked for 10am and was in the chair at 10:05. The stylist understood exactly what I wanted.", Language: "en",
			PricePaid: f(1200), WouldRecommend: b(true), ReturnLikelihood: i(5),
			DiscoverySource: "instagram", ExpectationMatch: "as_expected",
			CriterionScores: map[string]int{"skill_result": 5, "cleanliness": 5, "time_respect": 5, "price_transparency": 4, "appointment_reliability": 5}}},
		{3, reviews.Input{TargetID: selam.ID, OverallRating: 2, Title: "Case did not match photos",
			Body: "Ordered a phone case from their Telegram channel; the delivered one was a cheaper model than the photo.", Language: "en",
			PricePaid: f(600), WouldRecommend: b(false),
			DiscoverySource: "telegram", ExpectationMatch: "very_different",
			CriterionScores: map[string]int{"product_matched_description": 1, "seller_communication": 3, "delivery_speed": 4, "refund_handling": 2}}},
		{4, reviews.Input{TargetID: selam.ID, OverallRating: 4, Title: "Fast delivery",
			Body: "እቃው በሰዓቱ ደረሰ፣ እንደፎቶው ነው። ዋጋውም ተመጣጣኝ ነው። እመክራለሁ።", Language: "am",
			PricePaid: f(950), WouldRecommend: b(true),
			DiscoverySource: "telegram", ExpectationMatch: "as_expected",
			CriterionScores: map[string]int{"product_matched_description": 4, "delivery_speed": 5, "price_fairness": 4, "seller_reliability": 4}}},
		{0, reviews.Input{TargetID: tomoca.ID, OverallRating: 5, Title: "Best macchiato in Addis",
			Body: "ማኪያቶው ሁሌም አንድ አይነት ጥራት አለው። ቦታው ትንሽ ነው ግን ታሪካዊ ነው።", Language: "am",
			PricePaid: f(120), WouldRecommend: b(true), ReturnLikelihood: i(5),
			DiscoverySource: "friend",
			CriterionScores: map[string]int{"taste": 5, "price_fairness": 5, "service_speed": 4, "atmosphere": 4}}},
		{3, reviews.Input{TargetID: tomoca.ID, OverallRating: 4, Title: "Great beans to take home",
			Body: "Always busy but the queue moves fast. Bought beans for home and they were fresh.", Language: "en",
			WouldRecommend: b(true), ReturnLikelihood: i(4),
			DiscoverySource: "google_maps",
			CriterionScores: map[string]int{"taste": 5, "service_speed": 4, "portion_size": 3}}},
	}
	for _, or := range otherReviews {
		if _, err := newReview(or.user, or.in); err != nil {
			return err
		}
	}

	// Helpful votes and one business response, exercising those paths.
	for _, voter := range []int{1, 2, 3} {
		if _, err := reviewsRepo.Vote(ctx, firstKategnaReview.ID, customers[voter].id); err != nil {
			return fmt.Errorf("seeding helpful vote: %w", err)
		}
	}
	if _, err := bizRepo.CreateResponse(ctx, firstKategnaReview.ID, ownerID,
		"እናመሰግናለን! Thank you for visiting us — we look forward to serving you again."); err != nil {
		return fmt.Errorf("seeding business response: %w", err)
	}

	slog.Info("seed complete",
		"admin", cfg.SeedAdminEmail,
		"customers", len(customers),
		"targets", 5,
		"reviews", len(kategnaReviews)+len(otherReviews))
	return nil
}
