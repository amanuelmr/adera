# UI Inspiration Research — Global Review Platforms (2025–2026)

**Purpose:** Pattern research to inform Adera, the Ethiopian trusted review platform. Backend-first MVP; this document informs API/data-model decisions now and frontend design later.
**Date accessed for all sources: 2026-07-17.**
**Research method & honesty note:** Findings are based on web search results, platform help-center documentation, corporate trust pages, and secondary UX-analysis articles. The researcher could not screenshot or interactively browse live logged-in UIs, and one direct fetch (Trustpilot's review-labels help article) failed to render (JS-only page), so some details are corroborated from help docs and third-party write-ups rather than first-hand inspection. Items not fully verified are marked **[unverified detail]**. Treat exact pixel-level claims with caution; treat the structural/behavioral claims as reliable.

**Blanket legal rule (applies to every entry below):** Never copy proprietary code, assets, logos, brand names, mascots, icon sets, illustration styles, or exact page layouts from any platform. "Bubble ratings," "TrustScore," "Diners' Choice," "Elite Squad," "Travelers' Choice," "Verified Purchase" (as a trademark-styled badge), etc. are brand assets. Adopt the *interaction patterns and information architecture*, express them in original visual design and original terminology (ideally Amharic-first terminology).

---

## 1. Yelp

### 1.1 Review cards
- **Source:** Yelp Support Center / UX analyses of Yelp review cards — https://www.yelp-support.com/ ; https://bricxlabs.com/blogs/review-card-web-design-examples ; https://www.uxpin.com/studio/blog/review-card/
- **Observation:** Yelp's review card leads with the reviewer identity block (avatar, name, review count, photo count, Elite badge), then star rating + date, then long-form text, then attached photos, then reaction buttons. Reviewer credibility signals are as prominent as the review itself.
- **Adopt:** Reviewer identity block with contribution stats ("12 reviews · 4 photos") as a credibility signal; star rating and date on one line; photos attached inline. Backend implication (implemented): review responses join cheaply to reviewer aggregate stats.
- **Avoid:** Yelp's density of ads and CTAs around reviews; copying the Elite branding.

### 1.2 Helpful-vote mechanics (reactions)
- **Source:** Yelp Support — "How do I vote a review as useful, funny, or cool?" — https://www.yelp-support.com/article/How-do-I-vote-a-review-as-useful-funny-or-cool?l=en_US ; Yelp blog — https://blog.yelp.com/community/yelp-reveals-its-most-interesting-reviews-plus-new-in-app-review-reactions/
- **Observation:** Yelp migrated its classic "Useful / Funny / Cool" votes to four reactions: "Helpful," "Thanks," "Love this," "Oh no" (old votes were mapped into the new ones). Reactions are one-tap, per-review, per-user.
- **Adopt:** A single "Helpful" vote for MVP (one per user per review, reversible, counted, feeds ranking). Yelp's vote-type migration shows the backend should not paint itself into a boolean corner; Adera keeps votes in their own table so a `type` column can be added later.
- **Avoid:** Launching with 4 whimsical reaction types — it fragments the ranking signal; "Funny/Cool" style votes add little decision value.

### 1.3 Trust / moderation messaging (recommendation software)
- **Source:** Yelp Trust & Safety — https://trust.yelp.com/recommendation-software/ ; https://www.yelp-support.com/article/Why-would-a-review-not-be-recommended?l=en_US
- **Observation:** Yelp publicly explains that automated software evaluates every review on hundreds of signals across four categories (conflicts of interest, solicited reviews, reliability, usefulness). Non-recommended reviews are *not deleted*: they move to a "not currently recommended" section, are excluded from the star average and count, and status can change over time. Yelp hosts a dedicated public Trust & Safety site explaining this in plain language.
- **Adopt:** The single most important pattern for a "trusted" Ethiopian platform: (a) a review state machine (`pending / published / under_review / rejected / hidden / removed`) rather than binary delete; (b) hidden reviews excluded from aggregates but preserved; (c) a public plain-language moderation policy page in Amharic and English (see docs/moderation-policy.md).
- **Avoid:** Yelp's opacity backlash — businesses widely complain legitimate reviews get hidden with no recourse (https://thriveagency.com/news/why-yelp-is-hiding-so-many-legitimate-online-reviews-in-its-non-recommended-section/). Provide a reviewer-facing explanation and appeal path from day one.

---

## 2. Google Maps Reviews

### 2.1 Rating distribution & filtering
- **Source:** GMB Everywhere rating-distribution analysis — https://www.gmbeverywhere.com/features/analyze-rating-distribution ; OpenWeb Ninja Google reviews API docs — https://www.openwebninja.com/api/google-maps-reviews
- **Observation:** Google shows a 5-row horizontal bar histogram next to the average; each bar is tappable and acts as a filter. Sort options: most relevant, newest, highest, lowest. Reviews can also be filtered by auto-extracted topic chips and searched by keyword.
- **Adopt:** Tappable histogram bars = filters (one control does double duty — great for small screens); sort enum of `relevant | newest | highest | lowest | most_helpful`; distribution counts exposed in the target stats API response so the frontend gets histogram + average + count in one payload (implemented in `GET /api/v1/targets/{id}/stats`).
- **Avoid:** Google's "most relevant" default without explanation — undisclosed ranking breeds distrust; label the default sort and let users change it.

### 2.2 Review card anatomy & attributes
- **Source:** OpenWeb Ninja API field list — https://www.openwebninja.com/api/google-maps-reviews ; Google Maps support threads — https://support.google.com/maps/thread/328305069
- **Observation:** Each review carries: reviewer name/photo, Local Guide level + contribution count, star rating, relative date, text, photos, structured attributes (service type, meal type, price chips), owner response with its own timestamp, and a helpful count. Rating-only reviews are allowed.
- **Adopt:** Relative dates (frontend concern; backend returns RFC 3339); structured attribute data per category (Adera: category criteria + discovery source + price paid); owner response stored as a distinct entity with its own timestamps (implemented).
- **Avoid:** Google's anonymous-feeling "A Google user" fallback; Adera requires a display name. Adera also requires review text (min 20 chars) because a trust-focused platform with a small corpus needs substance; revisit if submission volume suffers.

### 2.3 Owner responses & reporting
- **Source:** Google Business Profile Help — https://support.google.com/business/answer/4596773?hl=en ; https://support.google.com/contributionpolicy/answer/7445749?hl=en ; https://yourcx.io/en/blog/2025/12/best-practice-response-time-for-google-maps-reviews/
- **Observation:** One official response per review, publicly labeled, editable. Reporting: reason picker → send; evaluation takes days to ~3 weeks. Cited research: 97% of review readers also read owner responses; 53% expect a reply to a negative review within a week.
- **Adopt:** One-response-per-review model (simple, prevents flame wars) — implemented as a unique constraint; report flow = fixed reason enum + free-text detail + status visible to the reporter; business dashboard stats endpoint.
- **Avoid:** Google's long, silent moderation turnaround — in a trust-scarce market, acknowledge reports instantly and show status (implemented: reports have status and are listable by the reporter).

---

## 3. Trustpilot

### 3.1 Review card anatomy & verification labels
- **Source:** Trustpilot Help — https://help.trustpilot.com/s/article/About-Trustpilots-review-labels?language=en_US (fetch failed — JS-rendered; corroborated via https://wiserreview.com/blog/trustpilot-reviews/ and https://help.trustpilot.com/s/article/TrustScore-and-star-rating-explained?language=en_US) **[partially unverified detail]**
- **Observation:** Review cards show reviewer name + country + review count, star row, a **"Verified"** label when the review is tied to a proven transaction, review title + body, and a distinct **"Date of experience"** separate from posting date. Unverified reviews still count but lack the label.
- **Adopt:** Two-date model (`experience_date` vs `created_at`) — implemented; a `verification_level` enum rendered as a labeled badge with tap-to-explain; review title field.
- **Avoid:** The "TrustScore" word-mark and green-star trade dress; Trustpilot's most-criticized dynamic — businesses paying for tools that shape which reviews get invited (perceived pay-to-play).

### 3.2 Aggregate score design
- **Source:** https://help.trustpilot.com/s/article/TrustScore-and-star-rating-explained?language=en_US
- **Observation:** TrustScore (1.0–5.0, one decimal) is not a plain mean: it weights recency and volume and uses a Bayesian-style prior so few-review businesses aren't extreme.
- **Adopt:** Bayesian confidence-adjusted **ranking** score computed server-side and documented publicly (docs/rating-and-ranking.md) — but Adera shows the **raw average, unaltered, right next to it** with the count ("4.3 · 128 reviews").
- **Avoid:** Hiding the fact that a ranking score isn't a simple mean; that discovery erodes trust when users compute it themselves.

### 3.3 Trust/transparency messaging
- **Source:** Trustpilot Trust Report 2025 — https://corporate.trustpilot.com/trust/trust-report-2025 ; https://cdn.trustpilot.net/trustsite-consumersite/trustpilot-transparency-report-2024.pdf
- **Observation:** Annual public Trust Report with hard numbers (4.5M fake reviews removed in 2024, 90% automatically); flagged reviews remain visible while under investigation; the flag flow tells the flagger what happens next.
- **Adopt:** Even at small scale, publish simple moderation stats (reviews received/removed/reasons) — the moderation_actions audit table makes these counters trivial; reports get a trackable status.
- **Avoid:** Overclaiming AI moderation Adera doesn't have; MVP messaging is "reviewed by our team," not "AI-powered."

---

## 4. Tripadvisor

### 4.1 Submission form & category sub-ratings
- **Source:** https://www.tripadvisor.com/business/insights/resources/bubble-rating ; https://www.tripadvisor.com/Trust-lvBd3L1aU38Y.html ; https://www.guesttouch.com/blog/understanding-tripadvisors-new-rating-updates-a-quick-guide-for-hoteliers
- **Observation:** Overall 1–5 rating with labeled points ("Terrible" → "Excellent"); *optional* category sub-ratings that change by property type; trip metadata (traveler type, month of visit); title required; since early 2025 the aggregate displays to the nearest tenth.
- **Adopt:** Category-specific sub-ratings driven by a database `category → criteria[]` table (the key schema decision — implemented as `category_criteria` with per-criterion required flag, scale, translations); visit-context metadata (`experience_date`, `discovery_source`); tenth-precision aggregates.
- **Avoid:** "Bubbles" (trademark-adjacent); making all sub-ratings required; long minimum text requirements that suppress submissions in a nascent market (Adera minimum: 20 characters).

### 4.2 Trending / awards
- **Source:** https://www.tripadvisor.com/TravelersChoice ; https://www.tripadvisor.com/Trust-lAkEadpFVLyU.html
- **Observation:** Annual award lists computed from review quality + quantity over a fixed 12-month window, with a "Trending" subcategory for places rising fast.
- **Adopt:** Periodic computed lists ("Top rated," "Trending" = recent activity + recent score) from a transparent formula over a fixed window — implemented as `GET /api/v1/targets/trending` and `/top-rated` with the formula documented in docs/rating-and-ranking.md.
- **Avoid:** Award names that echo "Travelers' Choice"; opaque criteria.

---

## 5. G2

### 5.1 Structured review form
- **Source:** https://research.g2.com/research-guidelines ; https://documentation.g2.com/docs/research-scoring-methodologies ; https://qondor.com/blog/your-3-step-guide-to-g2-software-reviews-and-why-your-review-matters
- **Observation:** The most structured form studied: identity verification first, then sectioned questions — required "What do you like best?/dislike?" boxes, numerical criteria ratings each with an N/A option, and use-case context. Key criteria are weighted higher in the satisfaction score.
- **Adopt:** **N/A option on every criterion** (implemented: criterion scores are optional unless the criterion is marked required); reviewer-context fields appropriate to Ethiopia; criteria answers stored as rows (`review_criterion_scores`), not columns.
- **Avoid:** G2's 10+ minute form length — for consumers, required fields stay minimal (overall stars + text); everything else optional/progressive.

### 5.2 Verification badges
- **Source:** https://www.team4.agency/glossary/what-are-g2-reviews-and-how-do-they-work ; https://slashexperts.com/post/understanding-g2-review-guide/
- **Observation:** Tiered verification: identity check to submit at all; optional screenshot proof earns a "Verified Current User" badge that visibly upgrades weight.
- **Adopt:** Tiered trust levels on the review entity — implemented as `verification_level`: `unverified → media_attached → receipt_submitted → location_verified → partner_verified`, designed now even though MVP grants only the first three.
- **Avoid:** Requiring LinkedIn/business email — wrong for the Ethiopian consumer context; phone OTP is the local analog.

---

## 6. OpenTable

### 6.1 Verified diner model & dining criteria
- **Source:** https://help.opentable.com/s/article/Ratings-and-Reviews-1505261056054?language=en_US ; https://www.opentable.com/restaurant-solutions/resources/opentable-diners-choice/
- **Observation:** Reviews only after a completed reservation — every review is "verified diner" by construction. Post-visit prompt asks overall stars plus **Food, Service, Ambiance, Value**; per-criterion averages are displayed on the restaurant page.
- **Adopt:** Proven minimal dining criteria set — folded into Adera's restaurant criteria (taste, service speed, atmosphere, price fairness…); per-criterion averages exposed on the target stats endpoint.
- **Avoid:** Closed-loop-only reviews at MVP — Adera doesn't own transactions initially, so a pure OpenTable model yields zero reviews. Hybrid instead: open reviews + higher-trust labels for verifiable ones.

### 6.2 Diners' Choice (trending)
- **Source:** https://help.opentable.com/s/article/What-are-OpenTable-Diners-Choice-lists-1505260081693?language=en_US
- **Observation:** Monthly-refreshed lists per city/neighborhood, sliced by criterion, computed from verified feedback.
- **Adopt:** Per-area lists ("Best value in Piassa") are cheap to compute from criterion scores — deferred but the schema (criterion scores keyed by area-scoped targets) supports it.
- **Avoid:** The "Diners' Choice" name.

---

## 7. Letterboxd

- **Source:** https://www.wix.com/studio/blog/letterboxd-ui-is-changing-movie-reviews ; https://www.fivestarinsider.com/letterboxd-how-to-log-a-film/ ; https://letterboxd.com/films/popular/this/week/
- **Observation:** Half-star increments; review cards designed to be *screenshot-shareable*; logging separates recording (date, tags) from reviewing (text optional); "Popular this week" is simple activity-volume trending.
- **Adopt:** The insight that a review card is a *shareable social object* — cards should look good screenshotted into Telegram (Ethiopia's dominant sharing channel); simple, legible activity-volume trending over algorithmic opacity.
- **Avoid:** Half-stars for MVP (complicates aggregates and accessibility); heart-likes as the ranking signal — hearts reward wit, "helpful" rewards information.

---

## 8. DoorDash (delivery-style ratings)

- **Source:** https://help.doordash.com/en-us/consumers/article/frequently-asked-questions-most-liked-items-item-ratings ; https://help.doordash.com/en-us/merchants/article/how-are-customer-reviews-collected-and-displayed ; https://www.restaurantbusinessonline.com/technology/doordash-adds-reviews-food-ratings-app
- **Observation:** Post-order flow triggers on next app open: simplified sentiment scale, one-tap **tag chips** ("Good Flavor," "Accurate Order," "Late Delivery"), then optional text; item-level thumbs produce "Most Liked" menu tags. Store rating shown as lifetime 1–5.
- **Adopt:** Tag-chip feedback is the middle ground between star-only and full text — Adera's structured criteria play this role in MVP; dish-level ratings are an explicitly deferred feature ("best kitfo in the area" is a killer local feature for phase 2).
- **Avoid:** Replacing stars with non-standard scales — DoorDash can afford it because it owns the transaction loop.

---

## 9. Amazon-style e-commerce reviews

- **Source:** https://www.amazon.com/gp/help/customer/display.html?nodeId=G8UYX7LALQC8V9KA ; https://www.aboutamazon.com/news/retail/amazon-customer-reviews-star-ratings ; https://www.aboutamazon.com/news/amazon-ai/amazon-improves-customer-reviews-with-generative-ai
- **Observation:** "Verified Purchase" label when Amazon confirms the purchase; displayed star average is **weighted**, not a raw mean; histogram rows filter on tap; helpful counts in natural language ("312 people found this helpful"); top-positive/top-critical pair shown side by side; LLM "Customers say" summaries with sentiment chips that link to supporting reviews.
- **Adopt:** The **top-positive + top-critical pair** (excellent fairness signal, computable from helpful votes — supported by sort options); verified-average shown separately from raw average (implemented in stats endpoint). Chips-link-to-evidence is the pattern to keep if summaries ever ship.
- **Avoid:** Weighted averages presented *as* the average (Adera never alters the displayed mean); AI summaries at MVP — especially for Amharic without evaluated NLP quality, a wrong summary damages the trust brand disproportionately.

---

## 10. Cross-platform pattern references

- **Rating distribution displays:** Smashing Magazine — https://www.smashingmagazine.com/2023/01/product-reviews-ratings-ux/ ; https://smart-interface-design-patterns.com/articles/reviews-and-ratings-ux/ — J-shaped distributions are natural; always show count next to average; each histogram row is a filter. **Adopt** all. **Avoid** decimal-free averages, hiding low-star counts.
- **Search/filter UX:** Algolia — https://www.algolia.com/blog/ux/search-filter-ux-best-practices ; https://www.uxpin.com/studio/blog/filter-ui-and-ux/ — combine search with filters; few relevant facets; applied-filter chips; bottom-sheet filters on mobile.
- **Form friction:** https://ventureharbour.com/form-design-best-practices/ ; https://www.zuko.io/blog/8-tips-to-optimize-your-mobile-form-ux — single column, one question per screen, inline validation, ≥44px targets.
- **Empty states:** https://www.eleken.co/blog-posts/empty-state-ux ; https://carbondesignsystem.com/patterns/empty-states-pattern/ — explain why it's empty, one clear CTA. Critical for a new platform where most businesses start at zero reviews.
- **Owner-response impact:** https://www.customerexperiencedive.com/news/reviews-business-response-build-customer-trust/729411/ ; GatherUp study — 55% rank owner response in top-3 trust factors. **Adopt:** owner replies are first-class MVP (implemented).
- **Rating trends over time:** https://appbot.co/features/app-rating-trends/ — recent average (last 90 days) beside lifetime average with a delta is a consumer-facing trust feature. **Adopt:** implemented in the Reality Check / stats endpoints. **[Consumer-facing trend modules on Yelp/Tripadvisor themselves could not be directly verified; pattern verified in review-analytics products.]**
- **Accessible star ratings:** https://elevenways.be/en/articles/star-ratings-simple-yet-surprisingly-complex-for-accessibility ; https://dev.to/grahamthedev/5-star-rating-system-actually-accessible-no-js-no-wai-aria-3idl ; https://www.w3.org/WAI/tutorials/forms/custom-controls/ — details in docs/frontend-handoff.md.
- **Low-bandwidth/Africa:** https://lioncapventures.com/blog/building-mobile-first-web-apps-african-markets ; https://launchpad.ng/resources/africa-low-bandwidth-design ; https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Caching — ~40% of Sub-Saharan connections still 2G/3G; details in docs/frontend-handoff.md.
- **Ethiopic script/typography:** https://fonts.google.com/noto/specimen/Noto+Sans+Ethiopic ; http://www.geez.org/Calendars/ ; https://github.com/andegna/calender — details in docs/frontend-handoff.md.
