# Market Research: Ethiopian Review / Directory / E-commerce Landscape

**Project:** Adera — trusted review platform backend for Addis Ababa
**Date of research:** 2026-07-18
**Method:** Web search + page fetches (18 queries/fetches). All observations are dated 2026-07-18. Where a platform could not be verified or appears inactive, this is stated explicitly rather than inferred.

---

## 1. AddisReview

**Status: no distinct, active consumer-review platform under this exact name could be verified.** Searches for "addisreview" / "AddisReview" surfaced only:

- **Addis Review (YouTube channel)** — https://www.youtube.com/c/addisreview — accessed 2026-07-18. A tech/gadget tutorial channel, not a business-review platform.
- **Addis Business Review (via HabeshaLink listing)** — https://habeshalink.com/addis-business-review/ — accessed 2026-07-18. Described as providing "honest, detailed, professional reviews of businesses across Ethiopia," but only found as a directory listing; no verifiable live product, review counts, or traffic data.
- **Review Addis (Instagram)** — https://www.instagram.com/review_addis_/ — accessed 2026-07-18. Instagram-only review account, i.e., social content, not a structured platform.
- **ADDIS Review (Facebook page)** — https://www.facebook.com/addissreview/ — accessed 2026-07-18. Facebook-page-only presence.

**Observation:** the "review brand" space in Addis is fragmented across social accounts using near-identical names, with no dominant structured product. Amharic and English are mixed informally.
**Adopt:** treat social accounts (TikTok/Instagram pages doing reviews) as the real incumbent competitors, not websites; plan for ingesting/linking social content to structured business records.
**Avoid:** assuming a web-first competitor exists to out-feature; the actual gap is structure, permanence, and searchability of reviews that today live in videos and posts.

Adjacent name-collision platforms found (useful comparables):

- **Ethio Review** — https://ethioreview.com/en — accessed 2026-07-18. The site itself is a JS app that returned only a title on fetch; search descriptions indicate: mobile apps (App Store/Google Play), "verified review" capture via **QR stands and NFC devices placed at the business**, category browsing across Ethiopian cities. No public review-density numbers found.
  **Adopt:** in-venue review capture (QR/NFC → "visited" signal) is a genuinely strong verification idea for a low-trust market; design the backend to support proof-of-visit review flags from day one.
  **Avoid:** shipping a web app so JS-dependent that it is uncrawlable/unindexable — Ethio Review's own homepage renders as an empty title to bots, which kills SEO in a market where discovery is everything.
- **Yegna Review** — https://www.yegnareview.com/ — accessed 2026-07-18. Bills itself "Ethiopia's first modern review platform." Fetch returned essentially only the page title (again a client-rendered SPA); no visible review counts, categories, or language options could be extracted. Appears early-stage/thin.
  **Adopt:** the positioning language ("trusted review community for local recommendations") confirms the market narrative Adera is betting on.
  **Avoid:** launching city-wide with near-zero content; an empty review platform is the norm here and is why none has won.

---

## 2. Hulunem

- **Source:** Hulunem — https://hulunem.com/ (direct fetch blocked with HTTP 403; details from search summaries of hulunem.com, /about-us/, /pricing/, /frequently-asked-questions/) — accessed 2026-07-18.
- **Observation:** The most complete directory incumbent found. Features: business listings (restaurants, hotels, services, events, retail), customer reviews and ratings, curated city guides, **English and Amharic**, free listing tier plus paid Growth/Pro tiers. Review policy: **no anonymous reviews, moderated content, business owners get response tools**. Pricing details: free plan = profile, map, 5 images, reviews, 3 category tags, message leads at 100 Birr/message; Growth = better ranking, 10 images, unlimited leads, social promotion; Pro = top search priority, homepage banner, custom design, plus a free 1-page business website with hosting. Verification stance: "prefer 1,000 verified listings over 10,000 outdated ones"; verified badge is bundled with **premium plans**. Hulunem has expanded sideways into markets data (markets.hulunem.com), tenders (tenders.hulunem.com), and a membership product (atenu.hulunem.com) — signs of a directory monetizing breadth rather than review depth. No evidence found of high review density; its listing pages for delivery apps (e.g., https://hulunem.com/place/ethiopia/addis-ababa/addis-ababa/zmall-delivery/) rank in search but no review volume was observable.
- **Adopt:** English+Amharic parity as a launch requirement; named (non-anonymous) reviews with moderation; business-owner response tooling; freemium listing model as a proven local monetization pattern.
- **Avoid:** (a) selling the "verified" badge as a paid feature — verification-for-pay undermines the trust promise Adera is built on; (b) breadth-first sprawl (tenders, forex, guides) before review density exists; (c) blocking crawlers/fetchers so aggressively that third parties and AI assistants cannot read your pages (Hulunem 403s generic fetches).

---

## 3. Yenoral

- **Source:** Yenoral — https://yenoral.com/restaurants/ and https://yenoral.com/contact-us/ — accessed 2026-07-18.
- **Observation:** Positions as "Trusted Reviews & Smart Search in Ethiopia," based in Addis Ababa. In practice the restaurant section is **editorial, not user-generated**: branded taglines per venue ("Love at First Bite"), category navigation (traditional/cultural, casual, international, romantic, family-friendly), **no visible user reviews, no visible rating counts, English only**, and many restaurant links inactive. It is a WordPress-style content site (category archive URLs like /category/restaurants/traditional-and-cultural/ozone-addis-restaurant/) wearing review-platform branding. Heavy social presence (FB, IG, TikTok, YouTube, X, Telegram).
- **Adopt:** the "smart filters by vibe / coffee quality / occasion" framing resonates locally — model structured attributes (vibe, occasion, price band) as first-class queryable fields, not free text.
- **Avoid:** editorial-only "reviews" presented as community ratings — it caps scale and credibility; broken/inactive detail pages; English-only content for an Addis mass-market product.

---

## 4. Ethiopian Yellow Pages and business directories

- **Sources (all accessed 2026-07-18):**
  - Ethiopian Yellow Pages — https://ethiopianyellowpages.com/ — primarily serves the **Ethiopian diaspora in the US** (DC metro), not Addis consumers.
  - Ethiopia YP Online — https://ethiopiayponline.com/ — B2B marketplace claiming ~300,000 Ethiopian companies; bulk-imported directory data.
  - EthYP — https://www.ethyp.com/ — generic local business network/directory.
  - EthioVisit directory — https://www.ethiovisit.com/directory/addis-ababa/ — ~3,178 registered businesses across 180 cities; name/address/phone-level data only.
  - AddisBiz — https://addisbiz.com/business-directory/ — directory + yellow pages + "business reviews and product catalog"; long-running local player.
  - Adrasha — https://adrasha.com/ — industry-vertical business directory (automotive, construction, health, etc.).
  - AddisMap — https://www.addismap.com/ — interactive Addis city guide; notable for **paying users 10 ETB per 10 full reviews** as a review-bootstrapping incentive.
  - 2244.et — **could not be verified; no information found in search.** Stated explicitly: unknown/possibly defunct or unindexed.
- **Observation:** the directory layer is crowded but shallow: listings are contact-data dumps, review functionality is either absent or a checkbox feature with negligible density, verification is self-reported, search is category/keyword-only, Amharic support is inconsistent, and none show evidence of businesses responding to reviews. Several (ethiopiayponline's "300k companies") are clearly scraped/imported registries, not living profiles.
- **Adopt:** seed the business database from registry-style data the way directories do (it solves cold-start for *listings*), then differentiate on review density and freshness; AddisMap's micro-incentive for reviews is a proven local tactic worth a more fraud-resistant version.
- **Avoid:** competing as "yet another directory" — contact-info listings are commoditized; paying flat cash per review without proof-of-visit (invites farming); diaspora-oriented positioning when the target is Addis residents.

---

## 5. Jiji Ethiopia

- **Sources (accessed 2026-07-18):**
  - Jiji Ethiopia — https://jiji.com.et/ — ~718,931 live classified ads.
  - Jiji Premium Services launch — https://www.einpresswire.com/article/721196094/ — ~25k active sellers, ~500k+ monthly users in Ethiopia (as of June 2024); #1 in cars and real estate.
  - JustUseApp reviews — https://justuseapp.com/en/app/1525181998/jiji-ethiopia/reviews — safety score 4.5/100 from 218 analyzed reviews; users describe it as "a breeding ground for fraudsters," "a swamp of scams and fake listings," a marketplace for stolen goods, with "nonexistent" customer support.
  - Shega review of Jiji — https://shega.co/news/review-of-the-top-3-online-marketplaces-in-ethiopia-jiji — accessed 2026-07-18 (fetch blocked; search summary only).
- **Observation:** Jiji is the largest classifieds player by inventory and traffic, yet consumer trust is catastrophically low. Its trust tooling is thin: a blue-check "verified seller" badge, seller history, and advice to meet in public. **No product-level reviews, no transaction rail, no dispute resolution, no hype-vs-reality mechanism.** This is the clearest proof in-market that traffic without trust does not solve the consumer problem.
- **Adopt:** seller/business reputation as a persistent, visible track record; safety education content as an SEO/trust asset.
- **Avoid:** listings-first growth with moderation as an afterthought; unmoderated free posting; "verification" that is a paid or cosmetic badge rather than an evidence-backed status.

---

## 6. E-commerce / delivery platforms

- **Sources (accessed 2026-07-18):**
  - Zmall — https://www.zmalldelivery.com/ and https://wanderlog.com/place/details/8744978/zmall-delivery-service — full-city delivery (food, supermarkets, packages, ticketing, ~30-min promise); reviewers say service **declined**: 2+ hour delays, cold/spilled food, poor order-status communication.
  - Deliver Addis — https://apps.apple.com/us/app/deliver-addis/id1503459669 and https://play.google.com/store/apps/details?id=com.deliveraddis.deliveraddis — self-branded "#1 delivery service"; app rated ~3.02/5 (380 ratings); complaints: prepaid orders not refunded when items unavailable, refunds taking 5+ days, drivers demanding extra cash off-app.
  - beU Delivery — https://hulunem.com/place/ethiopia/addis-ababa/addis-ababa/beu-delivery/ and https://www.ethiovibes.com/travel/top-food-delivery-services-in-addis-ababa — Chinese-owned, price/speed-led, "all in one app" positioning; fast growth.
  - Tracxn Ethiopia food tech — https://tracxn.com/d/explore/food-tech-startups-in-ethiopia/ — 12 food-tech startups incl. beU, Zmall, Tikus, Deliver Addis, Metahu Addis.
  - Qefira shutdown — https://addisfortune.news/digital-classifieds-pioneer-qefira-shuts-down-under-new-ownership and https://shega.co/news/prominent-ethiopian-online-marketplace-qefira-shuts-down-following-acquisition — Qefira (~400k monthly visitors, ad-revenue model) closed June 2023 after Ringier consolidated ROAM. A cautionary tale: audience without a durable business model dies.
  - Engocha — https://engocha.com/ — free classifieds/marketplace, still active. HellooMarket — https://helloomarket.com/ — niche catalog e-commerce (leather, coffee, apparel).
- **Observation:** delivery apps have order flows but **weak or captive review systems** (in-app ratings are not public, not searchable, and don't discipline restaurants); refund/dispute handling is the #1 complaint across all of them; none address restaurant discovery quality. Product-level reviews effectively do not exist anywhere in Ethiopian e-commerce.
- **Adopt:** integrate/ingest "order fulfilled" style signals where partnerships allow, as high-quality verified-experience events; treat refund/dispute pain as a review dimension ("does this business make refunds right?") no incumbent captures.
- **Avoid:** building delivery/transactions into the MVP; ad-only revenue dependence (Qefira); captive ratings that never surface publicly.

---

## 7. Google Maps in Ethiopia

- **Sources (accessed 2026-07-18):**
  - The Reporter, "Lost In Translation: Ethiopia's Google Maps Navigation Challenge" — https://www.thereporterethiopia.com/37719/ — even in Addis, Google Maps misplaces businesses and mislabels institutions; data depends on volunteer contributors and slow update pipelines; street/name changes lag badly.
  - Medium, Tewodros Hailegeberel — https://medium.com/@Teddyumd/how-i-created-digital-map-for-addis-ababa-when-google-map-was-not-good-enough-1efd3e0c360 — practitioner account of building a local map because Google's Addis coverage was inadequate.
  - Top-Rated.Online Addis ranking — https://www.top-rated.online/countries/Ethiopia/cities/Addis+Ababa/top-rated — review mass concentrates on a small head of famous places; the long tail of cafés/salons/shops is thin.
  - TikTok discover page "Google Map Review Online Job Ethiopia" — https://www.tiktok.com/discover/google-map-review-online-job-ethiopia — evidence that "paid Google review" gig schemes are being marketed to Ethiopians, i.e., review-fraud supply exists locally.
- **Observation:** Google Maps is the default but structurally weak in Addis: wrong pins, stale data, no Amharic-first UX, addresses that don't match how Addis people navigate ("behind Edna Mall"), no proof-of-visit, and businesses rarely respond to reviews.
- **Adopt:** landmark-relative address modeling (free-text directions field + structured landmark references) alongside lat/lng; community-correction workflows; assume Google is the incumbent to complement, not replace.
- **Avoid:** relying on Google Places data as ground truth for Addis; ignoring review-fraud gig economies when designing anti-abuse.

---

## 8. Tripadvisor coverage of Addis Ababa

- **Sources (accessed 2026-07-18):**
  - Tripadvisor Addis restaurants — https://www.tripadvisor.com/Restaurants-g293791-Addis_Ababa.html — ~338 restaurants listed for a city of 5M+; top venues hold ~498 reviews, but coverage collapses outside the tourist head.
  - https://www.tripadvisor.com/Restaurant_Review-g293791-d1214032-Reviews-Addis_Ababa_Restaurant-Addis_Ababa.html — reviews note venue decline vs "new emerging traditional restaurants," illustrating stale rankings.
  - Decade-old forum threads still surface — https://www.tripadvisor.com/ShowTopic-g293791-i9958-k3376017-Restaurants_update_good_and_bad-Addis_Ababa.html
- **Observation:** Tourist-eye coverage: a few hundred venues, English-only, hotel-adjacent bias, stale content ranking well, zero coverage of local-consumer categories (salons, electronics, tailors), zero Amharic. Business responses rare. It does not serve residents.
- **Adopt:** recency-weighted ranking so the last-6-months experience dominates; resident-oriented categories from day one.
- **Avoid:** tourist-first framing; letting old reviews accumulate rank without decay.

---

## 9. TikTok / Instagram food discovery culture in Addis

- **Sources (accessed 2026-07-18):**
  - Shega, "Meet the TikTok Creators Reshaping Addis Ababa's Restaurant Scene" — https://shega.co/news/how-tiktok-is-slowly-shaping-where-and-how-addis-ababa-eats (direct fetch 403; details via search extracts) — food-creator review packages now cost **tens of thousands up to ~100,000 Birr** per placement; a single Zelela video (300k+ views) visibly redirected foot traffic; consumers now check TikTok before trying restaurants.
  - Zelela (creator → platform) — https://www.tiktok.com/@zelelaapp — 200k+ followers; founder's stated policy "We go, we pay, and we eat" (refusing free meals) shows creators themselves see paid hype as the credibility threat.
  - TikTok discover pages — https://www.tiktok.com/discover/addis-ababa-food-review — dozens of active Amharic/English review accounts; content is vibes-and-price-focused (menus with Birr prices, landmark-relative locations).
- **Observation:** TikTok is the de facto review platform of Addis food culture — Amharic-native, price-transparent — but it is **pay-to-play (undisclosed sponsorship is normal), unsearchable, unaggregated, and has no accountability loop** when the hyped place disappoints. "Hyped vs reality" complaints are exactly the gap: there is nowhere structured to record that a viral spot didn't deliver.
- **Adopt:** social-content linking in the data model (URL attribution); disclosure flags for sponsored content; a "hype check" feature aggregating post-visit ratings; menu-item price capture as a future feature (TikTok reviews always quote prices).
- **Avoid:** fighting creators — they are acquisition channels; ignoring disclosure; text-only review UX long-term.

---

## 10. Telegram / Instagram / Facebook sellers — how people buy, scams, trust

- **Sources (accessed 2026-07-18):**
  - Awrari, "Safe Telegram Shopping in Ethiopia" — https://awrari.com/blog/safe-telegram-shopping-guide (fetch 403; search extracts) — Ethiopia's digital marketplace "lives on Telegram": no platform fees, direct seller chat, low data use; but **no buyer protection, no seller verification, no payment processing, no dispute resolution**.
  - The Reporter, "Telegram Shopping Scams On The Rise" — https://www.thereporterethiopia.com/26781/ (fetch 403; search extracts) — delivered goods far below advertised quality; stolen goods fenced via Facebook/Telegram.
  - TGStat Ethiopia sales channels — https://et.tgstat.com/sales — a large, measurable ecosystem of Ethiopian Telegram sales channels.
  - Shega marketplace roundup — https://shega.co/news/surprising-finds-three-ethiopian-digital-marketplaces-you-probably-havent-heard-about — churn of small marketplaces confirms fragmentation.
- **Observation:** the dominant purchase flow is: discover on TikTok/Instagram → chat on Telegram → pay via Telebirr/bank transfer or COD → hope. Trust signals are ad hoc and fakeable. Sellers keep no portable reputation; a burned scammer just opens a new channel.
- **Adopt:** **portable seller reputation** as a core entity — online sellers exist as review targets with social handles and no address; verified-purchase reviews via receipt evidence; scam-adjacent report reasons distinct from ordinary reviews.
- **Avoid:** requiring sellers to migrate off Telegram; treating social-seller reviews identically to physical-venue reviews (different verification evidence applies).

---

## 11. Internet / mobile landscape (context)

- **Sources (accessed 2026-07-18):**
  - DataReportal, Digital 2025: Ethiopia — https://datareportal.com/reports/digital-2025-ethiopia — 28.6M internet users (21.3% penetration); 85.4M mobile connections; 8.3M social identities (67.7% male — a gender skew that will shape early reviewer demographics); fixed median download ~9 Mbps.
  - Ethio Telecom subscribers passed 87M in H1 2025/26 — https://www.africanexponent.com/ethiopia-telecom-shifts-from-subscriber-growth-to-monetisation-and-infrastructure-sharing/
  - Safaricom Ethiopia — 12.2M 3-month-active customers (Feb 2026) — https://techafricanews.com/2026/02/02/safaricom-ethiopia-surpasses-12-million-users-as-m-pesa-adoption-grows/
  - Data price shocks — Safaricom raised data prices avg ~44%, some bundles +82% (Dec 2025) — https://addisinsight.net/2025/12/24/safaricom-ethiopia-raises-data-package-prices-by-up-to-82/ ; Ethio Telecom also raised prices — https://techpoint.africa/insight/techpoint-digest-1252/
- **Observation:** mobile-only, Android-dominant, **data is expensive and getting more so**, speeds modest, majority still offline. Any heavy web/app experience will be rationed by users.
- **Adopt:** brutal payload discipline (small API responses, aggressive image resizing, low pagination defaults); a Telegram bot as a near-zero-data client surface (future); SMS/USSD-tolerant verification flows.
- **Avoid:** video hosting at MVP (link out to TikTok instead); desktop-first anything.

## 12. Payments (context for trust/verification)

- **Sources (accessed 2026-07-18):**
  - Telebirr passed 50M users (Feb 2025), ~54.8M by Jul 2025; 110k agents, 60k merchants — https://ethiopianmonitor.com/2025/02/13/telebirr-subscribers-pass-50-million-mark/ and https://www.ethiotelecom.et/telebirr/
  - M-Pesa Ethiopia — 10.8M registered (Dec 2024) vs ~5.2M active (quarter ended Dec 2025, +258% YoY); EthSwitch integration connects 30+ banks/wallets — https://furtherafrica.com/2025/02/07/ and https://capitalethiopia.com/2026/04/05/ (registered-vs-active gap: treat "user counts" skeptically).
  - Payment fraud wave — SIM-swap-led account takeovers, fake-agent social engineering, 28 of 31 banks hit — https://scamwatchhq.com/ethiopia-scams-2026-telebirr-mobile-money-fraud/ ; fake Telebirr pages — https://pesacheck.org/scam-this-facebook-page-impersonating-telebirr-a-mobile-financial-service-is-fake/
- **Observation:** Telebirr is the universal rail and its receipts/transaction IDs are the most widely available proof-of-purchase artifact in the country. But phone numbers are takeover-prone (SIM swap), so phone possession ≠ durable identity.
- **Adopt:** phone-number (OTP) signup as baseline identity; Telebirr/M-Pesa receipt references as optional "verified purchase" evidence on reviews (private evidence, never public).
- **Avoid:** SMS-OTP as the *only* factor for high-privilege actions (business claims are moderator-reviewed for this reason); storing any payment credentials at MVP.

## 13. Consumer trust problems (fake goods, service no-shows, delivery scams)

- **Sources (accessed 2026-07-18):**
  - LivingEthio, electronics in Addis — https://www.livingethio.com/site/blog/where-to-find-electronics-in-addis-ababa — Merkato is the default source for cheap electronics; authenticity is buyer-beware. (Ethiopia-specific counterfeit-seizure statistics were **not found**; regional evidence from Kenya's Anti-Counterfeit Authority — https://www.aca.go.ke/media-center/news-and-events/163-mobile-stores-raided-213-fake-phones-seized-worth-over-ksh-10-million — confirms the East-African counterfeit-phone pattern. Honest gap: no hard Addis counterfeit data located.)
  - Salon/no-show economy — no direct reporting found on salon no-shows specifically (honest gap). Related: Addis salons book via Telegram/WhatsApp/phone (https://ajmenssalon.et/); a nascent salon-digitalization niche exists (https://www.digitalworldnetworks.com/addisababadigitalsalon), implying appointment reliability is unmanaged by software today.
  - E-commerce law — Electronic Transaction Proclamation No. 1205/2020 grants cancellation/refund rights and platform-operator obligations, but enforcement is missing: "law exists, compliance does not" — https://www.thereporterethiopia.com/51498/ and https://www.abyssinialaw.com/blog/legal-aspects-of-electronic-commerce-the-case-in-ethiopia
- **Observation:** trust failure is multi-category (goods authenticity, service reliability, delivery integrity, payment fraud) and legal recourse is theoretical. Reputation is the only enforcement mechanism realistically available to consumers.
- **Adopt:** category-specific review dimensions (electronics: authenticity/warranty; salons: appointment kept; delivery/sellers: refund handling — all implemented as seeded criteria); position Adera as the reputational enforcement layer the proclamation lacks.
- **Avoid:** promising dispute *resolution* (legal exposure, operational burden) — provide dispute *records* instead.

---

## Competitive summary

| Player | Review density | Verification | Amharic | Biz responses | Product-level reviews | Hype-vs-reality |
|---|---|---|---|---|---|---|
| Google Maps | Low outside head | None (fraud gigs exist) | Partial | Rare | No | No |
| Tripadvisor | ~338 restaurants, tourist head only | None | No | Rare | No | No |
| Hulunem | Low | Paid badge | Yes | Tools exist | No | No |
| Yenoral | None (editorial) | N/A | No | No | No | No |
| Ethio Review | Unknown/low | QR/NFC in-venue (novel) | Unknown | Unknown | No | No |
| Jiji | N/A (classifieds) | Cosmetic badge | Partial | N/A | No | No |
| Delivery apps | Captive, non-public | Order-based (captive) | Partial | No | No | No |
| TikTok creators | High engagement, unstructured | None; paid hype normal | Yes (native) | Via comments | Menu-item level, informally | Is the problem |

No incumbent combines: verified experiences + Amharic-first + resident categories + public business accountability + structured social-hype linkage. That is Adera's opening.

---

## Implications for the MVP backend

(Each item notes its implementation status in this backend.)

1. **Phone-number-first identity with real-name policy.** OTP-capable signup with Ethiopian number normalization; no anonymous reviews; display names required. *(Implemented: E.164 normalization, provider-abstracted OTP, display-name requirement. Second-factor for privileged actions: business claims require moderator review rather than SMS alone.)*
2. **Evidence-graded reviews as a core schema concept.** `verification_level`: unverified / media_attached / receipt_submitted / location_verified / partner_verified; verified averages exposed separately. *(Implemented.)*
3. **Bilingual content model from day one.** Category and criterion labels store translations (am/en seeded); reviews carry a language tag; search handles Ethiopic normalization and transliteration via alias tables. *(Implemented; Afaan Oromo label translations are a content task, not a schema change.)*
4. **Landmark-relative location model.** Free-text `address_text` ("behind Edna Mall") plus lat/lng plus area taxonomy. *(Implemented; structured landmark entities deferred.)*
5. **Business-response and profile-claim workflows in the MVP;** verification never purchasable. *(Implemented: claims with evidence + moderator decision; responses with edit audit.)*
6. **Recency-decayed ratings.** Recent (90-day) average exposed alongside all-time; trend surfaced in Reality Check. *(Implemented as recent-vs-historical comparison; half-life decay of the headline score deferred and documented.)*
7. **Social-hype linkage as a first-class feature.** Discovery source, expectation-match question, validated social links; no video hosting. *(Implemented. Creator profiles/sponsored flags deferred.)*
8. **Category-specific review dimensions.** *(Implemented: DB-driven criteria per category, seeded for all four categories including refund handling, appointment reliability, warranty, social-media accuracy.)*
9. **Online-seller entities distinct from physical venues.** *(Implemented: `online_seller` target type, online_only flag, social handles, no address required.)*
10. **Menu-item / offering-level records with prices in Birr.** *(Deferred; reviews capture voluntary price_paid in ETB as the first step.)*
11. **Moderation pipeline before growth features.** Rate limits per identity, review cooldowns, daily caps, report reasons incl. manipulated evidence, human queues with append-only audit logs. *(Implemented. Device fingerprinting and collusion heuristics deferred.)*
12. **Low-bandwidth API design as a hard requirement.** Small JSON payloads, cursor pagination (default 20/max 50), idempotent writes for flaky connections, and short-lived ETag caching for anonymous public reads. *(Implemented. Image derivatives deferred.)*
13. **Telegram bot as a second client surface.** *(Deferred; the JSON API is channel-agnostic by design.)*
14. **SEO-renderable content layer.** *(Frontend-phase requirement; recorded in docs/frontend-handoff.md — SSR strongly preferred.)*
15. **Incentives with fraud brakes, not cash-per-review.** *(Deferred entirely; documented as a risk if ever added. Cold-start plan: seed listings, concentrate on 2-3 neighborhoods and categories.)*
16. **Visible incentive and relationship disclosures.** *(Implemented: structured compensation and material-connection fields are returned on every review surface; `other` requires details. This records transparency but does not endorse paid-review acquisition.)*

---

*Honesty notes: (1) No active platform named exactly "AddisReview" could be verified — closest matches documented in §1. (2) "2244.et" returned no findable information. (3) Hulunem, Shega, Awrari, and The Reporter blocked direct page fetches (HTTP 403); their details come from search-engine extracts and should be spot-checked manually. (4) No Ethiopia-specific counterfeit-electronics seizure statistics or salon no-show reporting was found; flagged in §13 rather than filled with invented figures. (5) M-Pesa Ethiopia user figures conflict across sources (10.8M registered vs 5.2M active) — both cited with dates.*
