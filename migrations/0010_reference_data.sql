-- Reference data: the four launch categories with their review criteria,
-- cities, and Addis Ababa areas. Runs in every environment.
--
-- Category UUIDs are fixed so seed data and tests can reference them.
-- Note: "recommendation" and "likelihood of returning" from the product spec
-- are first-class review columns (would_recommend, return_likelihood) captured
-- for every category, so they are not duplicated as criteria rows.
-- Amharic labels are working translations pending native-speaker review.

INSERT INTO categories (id, code, name, name_translations, description, sort_order) VALUES
    ('11111111-1111-4111-8111-111111111101', 'electronics_repair', 'Electronics & Phone Repair',
     '{"am": "ኤሌክትሮኒክስ እና የስልክ ጥገና", "en": "Electronics & Phone Repair"}',
     'Electronics shops, phone sellers, and repair services', 1),
    ('11111111-1111-4111-8111-111111111102', 'beauty_salon', 'Beauty, Salon & Barber',
     '{"am": "ውበት፣ ሳሎን እና ፀጉር አስተካካይ", "en": "Beauty, Salon & Barber"}',
     'Salons, barbershops, and beauty services', 2),
    ('11111111-1111-4111-8111-111111111103', 'online_seller', 'Online Sellers & Delivery',
     '{"am": "የኦንላይን ሻጮች እና ዴሊቨሪ", "en": "Online Sellers & Delivery"}',
     'Telegram, Instagram, TikTok and web shops plus delivery services', 3),
    ('11111111-1111-4111-8111-111111111104', 'restaurant_cafe', 'Restaurants & Cafés',
     '{"am": "ምግብ ቤቶች እና ካፌዎች", "en": "Restaurants & Cafés"}',
     'Restaurants, cafés, and food spots', 4);

INSERT INTO category_criteria (id, category_id, code, name, label_translations, sort_order) VALUES
    -- Electronics & phone repair
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'product_originality', 'Product originality',
     '{"am": "የምርት እውነተኛነት", "en": "Product originality"}', 1),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'repair_quality', 'Repair quality',
     '{"am": "የጥገና ጥራት", "en": "Repair quality"}', 2),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'seller_honesty', 'Seller honesty',
     '{"am": "የሻጭ ታማኝነት", "en": "Seller honesty"}', 3),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'price_fairness', 'Price fairness',
     '{"am": "ተመጣጣኝ ዋጋ", "en": "Price fairness"}', 4),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'warranty', 'Warranty',
     '{"am": "ዋስትና", "en": "Warranty"}', 5),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'after_sales_support', 'After-sales support',
     '{"am": "ከሽያጭ በኋላ ድጋፍ", "en": "After-sales support"}', 6),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111101', 'repair_durability', 'Repair durability',
     '{"am": "የጥገና ዘላቂነት", "en": "Repair durability"}', 7),

    -- Beauty, salon & barber
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111102', 'skill_result', 'Skill and result',
     '{"am": "ችሎታ እና ውጤት", "en": "Skill and result"}', 1),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111102', 'cleanliness', 'Cleanliness',
     '{"am": "ንጽህና", "en": "Cleanliness"}', 2),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111102', 'time_respect', 'Time respect',
     '{"am": "ሰዓት አክባሪነት", "en": "Time respect"}', 3),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111102', 'staff_behavior', 'Staff behavior',
     '{"am": "የሰራተኞች አያያዝ", "en": "Staff behavior"}', 4),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111102', 'price_transparency', 'Price transparency',
     '{"am": "የዋጋ ግልጽነት", "en": "Price transparency"}', 5),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111102', 'appointment_reliability', 'Appointment reliability',
     '{"am": "የቀጠሮ አስተማማኝነት", "en": "Appointment reliability"}', 6),

    -- Online sellers & delivery
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'product_matched_description', 'Product matched description',
     '{"am": "ከማስታወቂያው ጋር መመሳሰል", "en": "Product matched description"}', 1),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'seller_communication', 'Seller communication',
     '{"am": "የሻጭ ግንኙነት", "en": "Seller communication"}', 2),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'delivery_speed', 'Delivery speed',
     '{"am": "የማድረስ ፍጥነት", "en": "Delivery speed"}', 3),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'packaging', 'Packaging',
     '{"am": "ማሸጊያ", "en": "Packaging"}', 4),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'price_fairness', 'Price fairness',
     '{"am": "ተመጣጣኝ ዋጋ", "en": "Price fairness"}', 5),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'refund_handling', 'Refund or return handling',
     '{"am": "ተመላሽ ገንዘብ አያያዝ", "en": "Refund or return handling"}', 6),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111103', 'seller_reliability', 'Seller reliability',
     '{"am": "የሻጭ አስተማማኝነት", "en": "Seller reliability"}', 7),

    -- Restaurants & cafés
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'taste', 'Taste',
     '{"am": "ጣዕም", "en": "Taste"}', 1),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'portion_size', 'Portion size',
     '{"am": "የምግብ መጠን", "en": "Portion size"}', 2),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'price_fairness', 'Price fairness',
     '{"am": "ተመጣጣኝ ዋጋ", "en": "Price fairness"}', 3),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'service_speed', 'Service speed',
     '{"am": "የአገልግሎት ፍጥነት", "en": "Service speed"}', 4),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'hygiene', 'Hygiene',
     '{"am": "ንጽህና", "en": "Hygiene"}', 5),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'atmosphere', 'Atmosphere',
     '{"am": "ድባብ", "en": "Atmosphere"}', 6),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'staff_behavior', 'Staff behavior',
     '{"am": "የሰራተኞች አያያዝ", "en": "Staff behavior"}', 7),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'menu_accuracy', 'Menu accuracy',
     '{"am": "የምናሌ ትክክለኛነት", "en": "Menu accuracy"}', 8),
    (gen_random_uuid(), '11111111-1111-4111-8111-111111111104', 'social_media_accuracy', 'Matched social media content',
     '{"am": "ከማህበራዊ ሚዲያ ይዘት ጋር መመሳሰል", "en": "Matched social media content"}', 9);

INSERT INTO cities (id, name, name_am, sort_order) VALUES
    ('22222222-2222-4222-8222-222222222201', 'Addis Ababa', 'አዲስ አበባ', 1),
    (gen_random_uuid(), 'Adama', 'አዳማ', 2),
    (gen_random_uuid(), 'Bahir Dar', 'ባህር ዳር', 3),
    (gen_random_uuid(), 'Hawassa', 'ሀዋሳ', 4),
    (gen_random_uuid(), 'Dire Dawa', 'ድሬ ዳዋ', 5),
    (gen_random_uuid(), 'Mekelle', 'መቀሌ', 6);

INSERT INTO areas (id, city_id, name, name_am, sort_order) VALUES
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Bole', 'ቦሌ', 1),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Piassa', 'ፒያሳ', 2),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Kazanchis', 'ካዛንቺስ', 3),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Megenagna', 'መገናኛ', 4),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Mexico', 'ሜክሲኮ', 5),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Sarbet', 'ሳር ቤት', 6),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'CMC', 'ሲኤምሲ', 7),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Gerji', 'ገርጂ', 8),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Summit', 'ሰሚት', 9),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Ayat', 'አያት', 10),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Merkato', 'መርካቶ', 11),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Arat Kilo', 'አራት ኪሎ', 12),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Sidist Kilo', 'ስድስት ኪሎ', 13),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Lideta', 'ልደታ', 14),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Jemo', 'ጀሞ', 15),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Lebu', 'ለቡ', 16),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Hayahulet', 'ሃያ ሁለት', 17),
    (gen_random_uuid(), '22222222-2222-4222-8222-222222222201', 'Old Airport', 'አሮጌው አውሮፕላን ማረፊያ', 18);
