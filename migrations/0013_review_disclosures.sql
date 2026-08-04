-- Public review disclosures for incentives and material connections.

ALTER TABLE reviews
    ADD COLUMN incentive_type text NOT NULL DEFAULT 'none'
        CHECK (incentive_type IN
               ('none', 'discount', 'free_product_or_service', 'payment',
                'contest_entry', 'loyalty_points', 'other')),
    ADD COLUMN material_connection text NOT NULL DEFAULT 'none'
        CHECK (material_connection IN
               ('none', 'current_employee', 'former_employee',
                'owner_or_executive', 'family_or_friend', 'business_partner', 'other')),
    ADD COLUMN disclosure_details text NOT NULL DEFAULT ''
        CHECK (char_length(disclosure_details) <= 500),
    ADD CONSTRAINT reviews_disclosure_other_details_check CHECK (
        (incentive_type <> 'other' AND material_connection <> 'other')
        OR char_length(btrim(disclosure_details)) >= 3
    ),
    ADD CONSTRAINT reviews_disclosure_details_relevance_check CHECK (
        incentive_type <> 'none' OR material_connection <> 'none'
        OR disclosure_details = ''
    );
