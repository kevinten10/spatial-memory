# Spatial Memory Network — Product Design Spec

**Date**: 2026-03-31
**Status**: Draft — Pending Implementation Plan
**Author**: Brainstorming session

---

## 1. Product Positioning & Core Experience

### One-Line Positioning

**"Pin memories to the real world. Discover them in person."**

**中文**："把记忆钉在真实世界里，让亲友和陌生人亲自到场才能发现。"

**对标**：空间版小红书 / Pinterest for the physical world

### Problem Statement

现有社交平台的内容在任何地方都能看到，与物理世界脱节。没有产品能同时做到：
1. 地理位置锚定的 AR 渲染（不是地图标记，是真正的空间 AR）
2. 持久化的个人多媒体记忆
3. 面向亲友和公众的授权分享机制

### Core Experience Loop

```
发现 → 感动 → 想留下自己的 → 创作 → 被别人发现 → 循环
```

**发现是核心驱动力**。用户必须先体验到"在真实世界某个位置看到别人的记忆"这个 magic moment，才会主动留下自己的记忆。留下记忆的创作行为可以通过其他平台导入内容完成——我们不需要成为用户最喜欢的"记录"工具，但要成为唯一的"空间发现"工具。

### Experience Principles

1. **到场即解锁** — 用户必须物理到达记忆所在位置（~50m 开始可见，15m 内完整查看）。这是核心差异化，永远不要妥协。
2. **渐进式呈现** — 远处看到光点/轮廓提示 → 走近逐渐清晰 → 到达位置完整展开。像游戏里的"战争迷雾"。
3. **不打扰，但引人好奇** — 不推送"你附近有记忆"。用户主动打开 app/戴上眼镜时，记忆自然浮现。
4. **情感优先于信息** — 这不是大众点评，而是个人情感和故事的载体。

### Content Visibility Model — Progressive Disclosure

| Layer | Visibility | Unlock Condition | Example |
|-------|-----------|-----------------|---------|
| **Private** | Creator only | Creator present at location | Personal diary-style memory |
| **Circle** | Authorized friends/family | Present + holds authorization token | "Dad, I watched the sunrise here last year" |
| **Public** | All users | Present at location | Landmarks, city stories, check-ins |
| **Featured** | All users (recommended) | Present at location | Editor's picks, high-engagement content |

### MVP Content Types

| Type | Priority | Notes |
|------|----------|-------|
| Photo + text caption | P0 | Lowest creation barrier |
| Short video (≤30s) | P0 | High immersion for memory scenes |
| Voice memo | P1 | Highest emotional warmth |
| Text-only | P1 | Poems, stories, messages |
| 3D model | P2 | AR-native but high creation barrier |
| AR graffiti | P2 | Fun but complex |

---

## 2. Technical Architecture

### System Architecture

```
┌─────────────────────────────────────────────────┐
│                   Client Layer                   │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Phone AR │  │Glasses AR│  │  Web Map View  │  │
│  │ (Phase1) │  │ (Phase2) │  │  (Phase1.5)   │  │
│  └────┬─────┘  └────┬─────┘  └──────┬────────┘  │
├───────┴──────────────┴───────────────┴───────────┤
│               AR Abstraction Layer               │
│  Unified spatial anchor interface                │
│  Shields ARCore/ARKit/MetaXR/Niantic differences │
├──────────────────────────────────────────────────┤
│                   API Gateway                    │
├──────┬──────────┬──────────┬────────┬────────────┤
│Spatial│ Content  │ Social   │Discover│ Moderation │
│Service│ Service  │ Service  │ Engine │  Service   │
├──────┴──────────┴──────────┴────────┴────────────┤
│                  Data Layer                       │
│  PostGIS │ Object Store (R2) │ Redis │ Vector DB  │
└──────────────────────────────────────────────────┘
```

### Core Services

**1. Spatial Service — "Where"**
- Stores geo-coordinates (lat/lng/alt) + orientation (yaw/pitch/roll) per memory
- PostGIS for spatial queries ("memories within 500m of me")
- Does NOT rely on AR platform Cloud Anchors for persistence — own coordinate data + dynamic local anchor reconstruction
- Outdoor: ARCore Geospatial API (Google Street View VPS)
- Indoor: Wi-Fi fingerprinting + BLE beacons (Phase 2)

**2. Content Service — "What"**
- Media storage: Cloudflare R2 (zero egress cost)
- Images: compressed original + thumbnail, CDN delivery
- Video: transcode to HLS adaptive streaming, ≤30s
- E2E encryption: private/circle memories encrypted client-side, server stores ciphertext
- Content hash deduplication

**3. Social Service — "Who can see"**
- User relationship graph (follow, friend circles, auth chains)
- Per-memory visibility settings
- Auth tokens: shareable encrypted tokens for circle memories, works without friend relationship

**4. Discovery Engine — "How to find"**
- Geo-fence trigger: mark memories as "discoverable" when user enters radius
- Heatmap ranking: when multiple memories at one location, sort by quality/recency/relationship closeness
- Vector similarity: interest-based recommendation of nearby public memories (Phase 2)

**5. Moderation Service — Safety**
- Public memories require review (AI screening + human review)
- Private/circle memories exempt but have report channel
- GLM-4V (ZhipuAI) for Chinese content understanding and moderation

### AR Abstraction Layer (Key Differentiator)

```
interface ISpatialAnchorProvider {
    createAnchor(lat, lng, alt, orientation) → AnchorHandle
    getLocalizationStatus() → { accuracy, source }
    getDevicePose() → { position, rotation }
    destroyAnchor(handle)
}

// Phase 1
class ARCoreGeospatialProvider implements ISpatialAnchorProvider
class ARKitLocationProvider implements ISpatialAnchorProvider

// Phase 2
class MetaXRProvider implements ISpatialAnchorProvider
class NianticVPSProvider implements ISpatialAnchorProvider
```

This abstraction decouples from any single AR platform and enables future migration to glasses SDKs.

### Phase 1 Technology Choices

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Mobile framework | Flutter + native AR bridge | Cross-platform efficiency, AR via native code for performance |
| AR positioning | ARCore Geospatial API (Android) + ARKit Location Anchors (iOS) | Global outdoor coverage, sufficient precision |
| Backend language | Go or Node.js | High-concurrency geo-query scenarios |
| Database | PostgreSQL + PostGIS | Spatial queries are core; PostGIS is industry standard |
| Object storage | Cloudflare R2 | Zero egress fees — critical for media-heavy app |
| CDN | Cloudflare | Seamless R2 integration |
| Cache | Redis + GeoHash | Fast hot-zone memory lookups |
| AI moderation | GLM-4V (ZhipuAI) | Strong Chinese content understanding, low cost |
| Real-time sync | WebSocket | Multi-user presence at same location (Phase 2) |

### Key Technical Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| GPS drift causes memory misplacement | High | Dual-layer positioning: GPS coarse → VPS visual fine. Allow user manual adjustment. "Fuzzy mode" shows area instead of precise pin when accuracy is low |
| GPS fails indoors | High | Phase 1 focuses on outdoor scenes (tourist spots, streets, parks). Indoor as Phase 2 |
| Too many memories at hotspots cause visual clutter | Medium | Spatial clustering + LOD (far: aggregate into one glow point, near: expand individuals) |
| E2E encryption vs content moderation conflict | Medium | Only public memories are moderated (plaintext). Private memories are E2E encrypted, exempt from moderation but have report channel |
| Multi-user sync at same location | Low | Phase 1 single-user only. Phase 2 adds shared sessions |

---

## 3. Business Model & Funding Story

### Revenue Phases

**Phase 1 (Month 0-12) — Free Growth**
- Zero revenue. All features free.
- Goal: 100K+ registered users, 1M+ geo-anchored memories

**Phase 2 (Month 12-24) — Payment Validation**

| Source | Model | Details |
|--------|-------|---------|
| Pro subscription | ¥18/month or ¥168/year | Free: 50 memory limit, 720p video. Pro: unlimited, 4K, 3D models, AR graffiti, advanced privacy |
| Brand memories | B2B partnerships | Venues/brands place branded content at locations. Per-location/time pricing |
| Memory gifts | Virtual goods | Special "memory containers" (time capsule, drift bottle, birthday cake shape). ¥6-30 each |

**Phase 3 (Month 24+) — Platform Ecosystem**

| Source | Model |
|--------|-------|
| Spatial ads | Location-based AR ads in commercial areas. CPM model |
| Open API | Third-party apps connect to spatial memory network. Usage-based pricing |
| Data licensing | Anonymized spatial behavior data to city planning, tourism boards |
| Glasses premium | AR glasses premium experience subscription when hardware arrives |

### Unit Economics Target (Phase 2)

```
Per-user monthly cost:
  Storage (avg 200MB/user)     ≈ ¥0.03
  CDN egress (R2 zero egress)  ≈ ¥0.00
  Compute (API calls)          ≈ ¥0.15
  AI moderation (public)       ≈ ¥0.05
  ─────────────────────────────
  Total                        ≈ ¥0.23/user/month

5% paid conversion × ¥18/month = ¥0.90 ARPU
Gross margin ≈ ¥0.67/user/month ✅ Positive
```

### Funding Pitch

**One-liner**: "We are the spatial memory network — users pin photos, videos, and sounds to physical locations in the real world. Others must visit in person to discover them. Phone AR first, AR glasses infrastructure when hardware arrives."

**Why now**:
1. ARCore Geospatial API matured in 2024 — global coverage, tech inflection point
2. Gen Z location-sharing behavior validated by Snapchat/Find My — user habits ready
3. AR glasses consumer launch imminent (2027) — accumulate spatial data now = own the future entry point
4. 18-24 month window before big tech moves in

**Competitive moats**:

| Moat | Description |
|------|-------------|
| Spatial data network effect | More memories per location → better discovery → more users → more memories. First mover wins |
| Emotional social graph | Not "follow" relationships but "I trust you with my most private memories." Extremely high switching cost |
| Presence verification | Patentable "must be physically present to unlock" core mechanic |
| Cross-platform spatial index | World's largest "what human memories exist at which physical locations" database |

**Comparable framing**:
- "空间版小红书 — Xiaohongshu lets you search for good content; we make you walk up to it"
- "Pinterest for the physical world — Pinterest pins inspiration to digital boards; we pin memories to the real world"

### Funding Cadence

| Round | Timing | Amount | Milestone |
|-------|--------|--------|-----------|
| Angel/Seed | 2026 Q3 | ¥3-5M | MVP live, seed city with 10K users, 100K memories |
| Pre-A | 2027 Q2 | ¥15-30M | 100K DAU, 1M+ memories, brand partnership validation, AR glasses demo |
| Series A | 2028 Q1 | ¥50-100M | AR glasses version live, 500K DAU, payment model validated |

---

## 4. Cold Start Strategy & MVP Roadmap

### The Triple Cold Start Problem

Spatial content platforms face three simultaneous requirements:
- Need content
- Need users
- Need content AND users at the SAME physical location

### Solution: "City Seed" Strategy

Focus on one city first, achieve critical content density, then expand.

```
Month 1:  Select seed city (recommend: Hangzhou — tourism + tech + founder familiarity)

Month 2:  Official team + 50 recruited "Memory Seed Officers" pre-plant content
          Cover 200 core locations (West Lake, Lingyin Temple, Hefang Street, malls...)
          5-10 curated memories per location = ~1,000 seed memories

Month 3:  Invite-only launch — "Hangzhou Memory Map"
          Target: local university students, travel bloggers, photography enthusiasts
          Hook: "Discover 1,000 memory easter eggs hidden across Hangzhou"

Month 4-6: User organic growth + expand to second city
```

**Why it works**: 200 locations × 5 memories = 1,000 memories. Users encounter one memory approximately every 500m in urban areas → sufficient discovery density.

### Viral Mechanisms

| Mechanism | Description |
|-----------|-------------|
| **Memory Invitation** | "I left you a memory at [location], come open it in person" → Share to WeChat. Recipient must visit to see → creates curiosity + physical action |
| **Memory Footprint Map** | Profile shows world map of "places I've left memories." Shareable to Moments. Like travel check-in maps but with emotional depth |
| **Crossed Memories** | Two people each leave a memory at the same location → system generates "You and XX have a crossed memory here" → social interaction trigger |
| **Memory Drift Bottle** | Random public memory pushed to nearby user: "Someone left a memory 300m from you. Want to go see?" |

### MVP Development Roadmap

```
Month 1-2: Infrastructure
├── Backend: PostGIS + R2 + API framework
├── Auth: phone number / WeChat login
└── Content pipeline: photo upload + storage + CDN

Month 3-4: Core Experience
├── AR positioning: ARCore Geospatial API integration
├── Spatial discovery: open camera → see nearby memory glow points
├── Memory viewing: approach → expand photo/video/text
└── Memory creation: capture → anchor to current location

Month 5: Social Layer
├── User relationships: follow, friend circles
├── Permissions: private/circle/public three-tier
├── Memory invitations: share to WeChat
└── Interactions: like, comment, bookmark

Month 6: Launch
├── Seed content pre-planting (200 locations)
├── TestFlight / internal beta
├── Seed city (Hangzhou) invite-only launch
└── Analytics instrumentation + core metric monitoring
```

### North Star Metrics

| Metric | Definition | 6-Month Target |
|--------|-----------|----------------|
| **Memory Discoveries** (North Star) | Times a user successfully arrives at location and views a memory | 50K/month |
| Memory Density | Memories per km² in seed city | Core area >20/km² |
| Creation Conversion | % of users who discovered a memory and then created one | >15% |
| Invitation Conversion | Received memory invitation → actually visited and opened | >8% |
| D7 Retention | New users still active after 7 days | >25% |

---

## 5. Team & Risk Management

### Minimum Founding Team (Phase 1)

| Role | Count | Core Responsibility | Key Skills |
|------|-------|-------------------|------------|
| Product/CEO | 1 | Product definition, fundraising, cold start ops | Product sense, fundraising, Unity/AR background |
| AR Engineer | 1 | ARCore/ARKit integration, spatial anchors, AR rendering | ARCore Geospatial API expert, 3D rendering |
| Backend Engineer | 1 | PostGIS spatial service, API, content pipeline | Go/Node + PostgreSQL + distributed systems |
| Mobile Engineer | 1 | Flutter/native app, camera, UX | Cross-platform + native AR bridging |
| Content Ops (part-time) | 1 | Seed content production, seed officer recruitment | Photography, community operations |

### AR Engineer Hiring — Critical Path

Most difficult and most important hire. Look for candidates from:
- Niantic alumni (Pokemon GO AR team)
- ByteDance/Kuaishou AR effects teams
- University researchers in SLAM/VPS

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Cold start failure — users find no memories | High | Fatal | City seed strategy + 20/km² density hard target. If unmet in 3 months, narrow to single commercial district |
| Poor AR positioning — memory drift | High | Severe | Dual-layer positioning + manual adjustment + "fuzzy mode" for low-accuracy areas |
| Big tech enters (Google/Apple/Snap) | Medium | Severe | Moat is data, not technology. Speed > perfection. Big tech builds general tools; we build emotional vertical |
| AR glasses delayed to 2029+ | Medium | Moderate | Phone experience must stand alone. Glasses are bonus, not requirement |
| Privacy compliance (GDPR/PIPL) | Medium | Severe | Privacy-by-design from Day 1: minimal data collection, E2E encryption, local processing. Hire legal counsel |
| Content safety (inappropriate content at public locations) | Medium | Severe | Mandatory moderation for public memories + geo-fencing (restrict public content near schools/government buildings) + rapid takedown |
| Business model doesn't work — users won't pay | Medium | Moderate | Phase 1 doesn't depend on revenue. If Pro conversion <2%, pivot to brand partnerships and spatial ads |
| Can't hire AR engineer | Medium | Severe | Founder builds prototype with ARCore first (has Unity/AR background), prove viability, then recruit |

### Biggest Non-Technical Risk: User Habit Formation

Current user mental model: "I arrive at a place → open Dianping/Xiaohongshu to search what's here"

Target mental model: "I arrive at a place → open [our app] to see what memories are here"

**Key insight**: Don't compete with search-intent products. Our trigger scenarios are:

| Trigger | User Motivation | Competition |
|---------|----------------|-------------|
| "My friend says she left something here for me" | Social curiosity | **None** |
| "Saw someone discover an amazing memory at West Lake on WeChat Moments" | FOMO + curiosity | **None** |
| "I want to surprise my boyfriend at the place we first met" | Emotional expression | **None** |
| "What interesting stories exist at this tourist spot" | Exploration | Xiaohongshu (weak overlap) |

The first three — socially-driven, emotionally-driven — are our exclusive territory.

---

## 6. Platform Strategy — Phone First, Glasses Upgrade

### Phase 1: Phone AR (2026 Q3 — 2027)
- ARCore Geospatial API (Android) + ARKit Location Anchors (iOS)
- Flutter app with native AR bridges
- Full feature set on phone

### Phase 1.5: Web Map View (2027 Q1)
- Browse memories on a map (no AR, just map pins)
- Low friction "window shopping" for non-app users
- Drives app installs ("download the app to see this memory in AR")

### Phase 2: AR Glasses (2027 — 2028)
- Meta Orion / future consumer AR glasses via Meta XR SDK
- Niantic VPS integration on Meta hardware
- Ambient discovery layer — glasses auto-highlight nearby memories
- Phone app remains for detailed viewing and creation

### Phase 3: Spatial Infrastructure (2028+)
- Open API for third-party apps
- "Spatial memory layer" as platform primitive
- Cross-device, cross-platform memory network

### AR Abstraction Layer ensures each phase addition is an implementation swap, not an architecture rewrite.

---

## 7. Market Context (Research Summary)

### Competitive Landscape

| Product | What It Does | Gap vs. Our Concept |
|---------|-------------|-------------------|
| ReplayAR | GPS-pins photos to locations | Single-user only, social sharing never shipped, appears stagnant |
| Snapchat Memories | Pins past Snaps to map locations | Private only, not true AR rendering, map overlay |
| Google Photos + Maps | Photo timeline + location | Pure 2D, no sharing of timelines |
| Apple Spatial Photos | 3D photo capture and viewing | $3,500 headset, no geo-anchoring |
| Niantic Lightship | Global VPS platform | SDK/platform, not consumer memory app |

**No product combines: geo-anchored AR + persistent personal memories + authorized social sharing.**

### Market Size

- Location-based services: $70.4B (2024) → $235B (2034), CAGR 12.6%
- AR market: $78.3B (2025) → $828.5B (2033), CAGR 34.3%
- Spatial computing: $168.6B (2025) → $897.5B (2035), CAGR 18.2%

### Smart Glasses Landscape (as of 2026)

No current consumer smart glasses simultaneously deliver: lightweight + spatial AR + available to buy.

- **Meta Orion** (prototype, ~98g): Best spatial AR demo in glasses form, consumer version expected 2027
- **Snap Spectacles 5** (226g, 45min battery): Best current spatial AR in glasses, dev kit only
- **Meta Ray-Ban** (49g): Travel-friendly but no AR display

Timeline: mainstream AR glasses expected 2027-2028 (Meta Orion derivative or Apple).

### Meta AR Ecosystem Status

- Meta shut down Spark AR (mobile) in Jan 2025, pivoting fully to headset/glasses
- Orion developer access: invitation-only, broader availability targeted 2026
- Consumer Orion: 2027 estimated
- **Meta has no public VPS/geospatial service** — use Niantic Lightship VPS or Google ARCore Geospatial API
- Niantic Spatial SDK v3.15 supports Meta Quest 3 including VPS
- Current best development path: Quest 3 passthrough MR (Unity 6 + Meta XR SDK), skills transfer to Orion

---

## Appendix: Key Decisions Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Relationship to TripMeta | Independent project | VR headsets too bulky for travel; smart glasses form factor needed |
| Core experience priority | Discovery > Creation | Users need to see magic first; creation can import from other platforms |
| Content model | Fully open UGC | Low creation barrier, maximum expression space |
| Visibility model | Progressive disclosure (default private, publishable to public, proximity unlock) | Privacy safety + growth potential + "must be present" ritual |
| Target platform | Meta AR glasses ecosystem | Betting on Meta Orion as 2027 consumer AR glasses winner |
| Product approach | Phone first, glasses upgrade | Phone AR is mature now; build user base and data moat before glasses arrive |
| Project goal | Startup venture | Full product planning + business model + fundraising story needed |
| Seed city | Hangzhou (recommended) | Tourism city + tech ecosystem + founder familiarity |
