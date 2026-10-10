# Evidence and placeholders

Facts cited by the economy specs, with the status of each one. See
[README.md](README.md) for the index of specs. A fact marked unverified comes
from a search snippet or general knowledge that was not opened or checked, and
must be verified before it is used to set a default.

Rows quote historical sources and keep their words (apprentice, master, guild,
craftsman, soldier). The specs use the neutral terms trainee, trainer,
association and skilled role for the same ideas.

| Fact | Source | Status |
|------|--------|--------|
| Welfare ratio is household income divided by the cost of a specified basket; 1 means baseline | Allen (2013) | read directly |
| Subsistence basket: 2 100 kcal per person per day; family of four, annual cost four times one person's basket | Allen (2013) | read directly |
| Annual income is day wage times 250 days in most of Allen's work | Allen (2013), Humphries and Weisdorf (2019) | read directly |
| Medieval labourers worked under 150 days a year; 1361-62 annual salary equal to 105 to 126 days; backward-bending labour supply (Dyer) | Humphries and Weisdorf (2019) | read directly |
| English 1700-60 labourers: London about 3, southern England about 2, northern England about 1.5, northern Italy about 1 | Allen (2013) | read directly |
| By 1820 about 2.5; by 1850 at least 3 | Allen (2013) | read directly |
| Humphries and Weisdorf's baskets-per-man figures divide by 3.25 to compare with Allen's original ratios | Humphries and Weisdorf (2019) | read directly |
| Calories cost: bread 2d, meat and cheese 3d, beer 4d, sugar 5d, oatmeal under 1d per 1 000 kcal | Allen (2013) | read directly |
| English labourer's wage in the 1780s about 2.9 grams of silver a day | Allen (2013), footnote 3 | read directly |
| Craftsman-to-labourer wage ratio in England (building trades): 1.56 (1220), 1.93 (1250), 1.93 (1300), 1.79 (1350), 1.50 (1400), 1.44 (1450), 1.47 (1500), 1.44 (1600), 1.60 (1700), 1.51 (1800), 1.52 (1850), 1.41 (1900) | Clark (2005), table A2, via Our World in Data CSV | read directly |
| 1344 household ordinance of Edward III: king's archers 6d a day, archers on horse 4d, archers on foot 3d, Welshmen on foot 2d | Wikipedia, "Yeoman archer", citing the ordinance | read directly (secondary); deferred, soldiers out of scope |
| Knight 2s, man-at-arms 1s, banneret 4s, armed man 6d a day; archer 6d "the same as a civilian craftsman" | search snippet, not a primary source | **unverified**; deferred |
| Per-person yearly basket quantities (about 165 kg bread or grain, 20 kg beans, 5 kg meat, 3 kg butter) | search snippet, not a primary source | **unverified placeholder** |
| Carpenters' wages about 3d a day in the late 13th century, 4d to 5d in the 14th, 5d or 6d in the 15th; a carpenter working 240 days at 5d to 6d earned about 6 pounds a year | search snippet of a journal article on English carpenters, 1200-1540 | **unverified** |
| Tool prices: anvil 20s, bellows 30s, hammers 8d to 2s 8d, two chisels 8d, a full set of armourer's tools 13 pounds 16s 11d | search snippet of a price list on a website (place and date unknown) | **unverified**, order of magnitude only |
| Guild fees: 1s to enrol an apprentice with carpenters, 3s 4d to join at the end of apprenticeship | same price list | **unverified**; deferred (training is out of scope) |
| Real rate of return on capital in England about 10% around 1300, falling to about 5% by 1400, and 4% to 5% before the Industrial Revolution | Clark, from land rents and rent charges, search snippets (not opened) | **unverified** |
| Anvils last decades to centuries with proper care; hammers wear out sooner. No use counts | search snippets of smithing articles and forums | **unverified**; durability values are assumptions |
| Carter rates of 5 pence per mile for cart or wagon hire, with the same charged for return mileage; a halfpenny per mile per hundredweight on wheeled vehicles | search snippet of an archival page (place and era unclear) | **unverified**; supports charging the round trip |
| Statute of Artificers 1563: apprentices bound for at least seven years, and a master limited to three apprentices | search snippets of history sites (Wikipedia and others, not opened) | **unverified** |
| Premiums for apprenticeship in mid-18th-century London: ordinary least squares elasticity with respect to journeymen's wages 0.35 and with respect to the master's set-up costs 0.25; after instrumenting wages (Wu-Hausman indicates endogeneity), the wage elasticity is about 1.0 and the set-up elasticity 0.23 to 0.25. The earlier "0.4" snippet is the rounded OLS wage value | Justman and Beek (2015), "Market forces shaping human capital in eighteenth-century London", Economic History Review 68(4), from Campbell's *The London Tradesman* (1747), <https://tzin.bgu.ac.il/~kvdbeek/papers/Campbell%20EcHR_2015.pdf> | **read** from text extracted from the PDF (abstract, the account of apprenticeship, the premium and cost discussion, the regression section); tables and appendices not read in full |
| Apprenticeship: usually seven years under an indenture; the master provided teaching, board and lodging; a cash premium was paid to the master at the start, typically 10 to 20 pounds; premiums cover the master's early net cost and the risk of early departure | same | read |
| Stylised-model results of round 3: the evidence rule revised to compare with the limit settles about 5.7% below the clearing price (from 11.3%); with labour in the unit cost, the seed over unit cost is 1.00 in the viability prototype; a cost floor of 100% freezes markets with a perishable and 70% does not; the default production halt turns a floor pin into a shortage cycle for inelastic flows | `spec-beliefs.md` 3.2.2 and 5.4, `spec-viability.md` 15, `spec-stability.md` 6.2 runs 1b, 2b and 5 | the workers' own Docker Python models; not validated against data; placeholders |
| An apprentice lived with the master, who supplied board, lodging and clothing; a premium was paid to the master | search snippets | **unverified** |
| Masters recouped training costs through apprentices working at below-market wages; medieval apprentices were not cheap labour; many apprenticeships ended early | search snippet of a research summary | **unverified** |
| Daily wages in pence for the labourer in the medieval period; loaf prices | not retrieved | **missing** |

Allen's 2001 paper and Munro's data tables were not retrievable. Real
calibration data should come from Allen's published data and Clark's day-rate
series (Clark 2007, table A2). Until then, ratios and the skilled-wage band are
the calibration, not pence.

Sources:

- Allen, R. C. (2013), "The High Wage Economy and the Industrial Revolution:
  A Restatement", Oxford Discussion Papers in Economic and Social History 115.
  <https://www.nuff.ox.ac.uk/economics/history/Paper115/allen115.pdf>
- Humphries, J. and Weisdorf, J. (2019), "Unreal Wages? Real Income and
  Economic Growth in England, 1260-1850", Economic Journal.
  <http://pinguet.free.fr/humphriesweisdorf.pdf>
- Clark, G. (2005), "The Condition of the Working Class in England,
  1209-2004", Journal of Political Economy 113(6), table A2. Series used:
  Our World in Data, "Wage of craftsmen relative to that of laborers in
  England, 1200-2000".
  <https://ourworldindata.org/grapher/wage-of-craftsmen-relative-to-that-of-laborers-in-england-1200-2000>
- "Yeoman archer", Wikipedia (1344 household ordinance pay rates).
  <https://en.wikipedia.org/wiki/Yeoman_archer>
- Clark, G., "Microbes and Markets: Was the Black Death an Economic
  Revolution?", on returns to capital from land rents (a search result, not
  opened). <https://faculty.econ.ucdavis.edu/faculty/gclark/papers/black1.pdf>


## Stability and macro-layer sources

Added with the stability review (`../roadmap.md`, "Stability") and
`../adr/0001-macro-economy-as-replaceable-layers.md`. These support design
direction, not numeric defaults. None sets a default.

| Fact | Source | Status |
|------|--------|--------|
| Firms compare inventory with recent demand to hire, fire and change prices; prices are bounded relative to marginal cost and change with a probability and a bounded step; goods trade daily while wages, prices and hiring are decided monthly; households have a hard liquidity constraint and spend a concave function of wealth; trading partners are local and re-chosen | Lengnick (2013), pages 3 to 7 | read directly |
| The same model has fixed numbers of households and firms (no entry or exit) | Lengnick (2013), the basic-properties part of the model description | read directly |
| The same model reproduces business cycles, a Phillips curve, a Beveridge curve and long-run neutrality of money | Lengnick (2013), introduction (a claim by the author; results not read) | read directly, results not read |
| Agent-based stock-flow consistent models with full balance sheets tend to a quasi steady state | Caiani et al. (2016), abstract and search summaries | **unverified** (summary only) |
| Zero-intelligence traders with a no-loss constraint reach high allocative efficiency; without the constraint they do not; neither reliably finds the equilibrium price | Gode and Sunder (1993) and Cliff, as summarised in search results | **unverified** (summary only) |
| Production delays cause price oscillations whose damping depends on lag length and demand elasticity | cobweb-model literature, search summaries | **unverified** (summary only) |
| Victoria 3 economic "death spirals" come from debt interest, falling tax efficiency and unprofitable production that stalls | one Steam forum thread | anecdote |
| Dwarf Fortress disabled its economy because it made the game worse | Wikipedia | read via summary |
| Pre-industrial economies tend to a Malthusian equilibrium when births rise and deaths fall with income | Clark, as summarised by VoxEU | **unverified** (summary only); superseded for the structure of the model by the rows below that read Clark directly |
| The Malthusian model rests on three assumptions: the birth rate is determined by custom and increases with living standards; the death rate declines as living standards rise; living standards decline as population increases (diminishing returns to a fixed land supply). In a stable population births equal deaths; the income at which they do (subsistence income) is fixed by the two schedules alone, independent of technology. A higher death schedule at a given income raises equilibrium income and lowers equilibrium population | Clark (2007), *A Farewell to Alms*, chapter 2, "The Logic of the Malthusian Economy" | read directly (text extracted with pdftotext; figures not seen). Supersedes the VoxEU summary row above |
| Surviving children per woman 1.93 to 2.10 for six European populations, c. 1300 to c. 1800 (Table 2.1) | Clark (2007), chapter 2 | read directly; no default set |
| "Gross fertility did not rise significantly with income": English testators c. 1620, richer ones (about ten times the assets) 12.7% against 7.8% never married, and gross fertility likely only 2% to 5% higher; "it does not do too much violence to reality to simply assume a given birth rate" | Clark (2006 draft of the book), chapter 4, draft pp. 90-93 | read directly (the October 2006 draft, not the published edition) |
| Across households income was a strong predictor of mortality: household income explained 62% of the variation in infant mortality across eight London parishes, 1538-1653; for the poorest groups death rates exceeded birth rates. In England 1540-1800 no association between national mortality and national income appears, attributed to shifts of the death schedule over time | Clark (2006 draft of the book), chapter 5, draft pp. 100-103 | read directly (draft; figures not seen) |
| Stylised-model results of POP: a Malthusian approach to the replacement ratio is monotone while the gain times the delay is below about 0.7 and cycles from about 1.5; a poorest-first allocation of deaths starves a pool-funded birth rule (86% fall in 80 years); a closed region's population is bounded by M / stake | `spec-population.md` 5.4, 5.5, 13, `models/pop/` | the worker's own Docker Python models; not validated against data; placeholders |
| EVE Online money enters through NPC bounties, mission rewards, NPC buy orders and insurance, and leaves through NPC sales, taxes and fees; mining is classified as production, not a money faucet | Engadget (2010); CCP monthly economic reports (2025 and 2026), via summaries | **unverified** (summary only) |
| CCP's 2010 quarterly economic newsletter | PDF is image-only; text could not be read | **missing** |
| abcEconomics has Firm and Household archetype classes, goods handled by the platform (a transfer moves the good), and message passing between agents | abcEconomics documentation, index page | read directly (index only) |
| ASSUME: a unit has a bidding strategy with `calculate_bids()` returning an order book; the same interface serves generators, demand and a custom unit; dispatch after clearing is the unit's own `execute_current_dispatch()` | ASSUME documentation, custom unit example | read directly |
| Labour supply curves with inflection points can create multiple, unstable equilibria (Friedman-Savage preferences) | Social Science and Economics Forum paper, 2014, abstract in a search snippet | **unverified** (snippet) |
| Victoria 3: orders sent to a market are scaled by the state's market access; a state with poor infrastructure relies more on local goods | Victoria 3 dev diary 9, search snippet; the page failed to load | **unverified** (snippet) |
| A Factorio recipe has lists of ingredients and results with amounts, a crafting time (`energy_required`), categories for machines, `main_product`, and per-result `probability`, `amount_min`/`amount_max`, `ignored_by_productivity` | Factorio `RecipePrototype` page, lua-api.factorio.com/latest/prototypes/RecipePrototype.html | opened; summary by the fetch tool, not read verbatim. Design inspiration only; sets no default |
| Victoria 3 buildings produce and consume goods per level scaled by workforce; production method groups include secondary outputs at reduced cost; methods unlock by technology or law | vic3.paradoxwikis.com/Production_methods | opened; summary by the fetch tool. Design inspiration only |
| abcEconomics `produce` takes a production function (Leontief, CES, Cobb-Douglas or custom) and input quantities, removes inputs, adds outputs, raises `NotEnoughGoods`; no production delay is documented | abce.readthedocs.io, Firm page | opened; summary by the fetch tool. Design inspiration only |
| ASSUME units have operating constraints such as efficiency and start-up times | search snippet | **snippet only; unverified; not used** |
| bazaarBot's algorithm as published: price beliefs as an interval, offers drawn uniformly in it, updates after each offer resolves, bid and ask update pseudocode with 1/10, 1/5, 1/6, 1/7 and 110%/120% factors, public window statistics, matching of highest bid to lowest ask with no crossing check in the pseudocode, replacement of bankrupt agents by the most profitable role | Doran and Parberry (2012), IJIGS 7(3), 35-48, <https://ianparberry.com/pubs/econ.pdf> | read in full (extracted text; figures not seen). The LARC-2010-03 report URL returned 404 |
| The paper's stability criteria (section VII): every commodity produced and consumed, none untraded for over 20% of rounds, mean volume above one unit a round, price variance 0.025 to 7.5 times the highest price, mean price change and its variance 0.02 to 0.9, fewer than 2 inflections a round; 0.19% of 1.5 billion random rule sets passed | same | read in full; the criteria are for prices of order one and several are scale-dependent |
| A perturbation of 10^-9 currency units in one agent changed some prices by about 20% after 500 rounds and bankrupted agents; attributed to floating point and compiler reordering | same | read in full |
| About 200 000 agent updates a second on one 3.2 GHz SPE, two floats per commodity per agent | same | read in full; a claim by the authors, not a measurement of this design (compare the placeholder in `spec-scale.md` 6.2) |
| Stockpiling "tends to dampen price swings"; price effects of a supply interruption arrive late as inventories run down; with no sales there are no price updates | same | read in full |
| Firm price rule: price considered up if inventory is below a lower bar (a multiple of the latest demand) and down if above an upper bar; price bounded between multiples of marginal cost (1 < phi_low < phi_high); new price = old * (1 +/- epsilon), epsilon uniform in [0, 0.02]; set with a probability below one; the economy breaks down only if price flexibility is too low | Lengnick (2013), the firm section, eqs. 6 to 10, Table 1, price-flexibility experiment | read from extracted text; Greek symbols lost; `vartheta` = 0.02 explicit; bound values 1.15 and 1.025 and probability 0.75 inferred, **unverified** |
| Cobweb model with naive expectations converges when demand is more elastic than supply at equilibrium (supply curve steeper) | Wikipedia, "Cobweb model" | opened via a summarising fetch; no formulas for adaptive expectations |
| Averaging expectations can stabilise a cobweb whatever the parameters, provided the demand slope is smaller than the supply slope | cobweb literature, search summary | **unverified** (snippet) |
| ZIP traders update a profit margin with a Widrow-Hoff rule (learning rate and momentum) | Cliff, search summary | **unverified** (snippet); the page returned 403 |
| ASSUME's custom unit example prices from the unit's marginal revenue and imposes no explicit bound on a bid | ASSUME documentation | read directly (that page); the bidding-agents page returned 404 |
| No default in `spec-production.md` or `spec-beliefs.md` is set from any source above | | all their numbers are placeholders |
| Household: liquidity `m_h` changes by income less spending and is restricted to positive values (a budget constraint); planned real consumption `c = (m / P)^alpha`, capped at `m / P`; with `0 < alpha < 1` consumption rises with wealth at a decaying rate; savings are precautionary; households trade only with a subset of firms and re-search by price and by unsatisfied demand; the reservation wage falls 10% after a month unemployed; `alpha` is explicit in equation 11 and Table 1 as 0.9 | Lengnick (2013), sections 2.1 to 2.4, Table 1 and Table 2 | read from text extracted from the PDF (`pdftotext`); Greek symbols partly lost; the household part, Table 1 and Table 2 read; the results sections not read in full. No default is set from it |
| `abcEconomics` `Household`: `consume(utility_function, input_goods)` and `create_cobb_douglas_utility_function(exponents)`; a household is production-free; `NotEnoughGoods` is raised when goods are short | <https://abce.readthedocs.io/en/master/Household.html> | opened through a summarising fetch |
| `abcEconomics` `Trade`: `sell` and `buy` commit goods and block them until settled; `accept` and `reject`; three subrounds (offer, answer, settle); an `Offer` has sender, receiver, good, quantity, price, status | <https://abce.readthedocs.io/en/master/Trade.html> | opened through a summarising fetch |
| ASSUME `BaseStrategy.calculate_bids(unit, market_config, product_tuples)` returns an order book; `calculate_reward(unit, marketconfig, orderbook)`; `BaseUnit.bidding_strategies` is a dict by market id; `set_dispatch_plan` and `execute_current_dispatch` give market feedback to the unit | ASSUME `assume/common/base.py`, <https://raw.githubusercontent.com/assume-framework/assume/main/assume/common/base.py> | opened; the fetch tool reproduced the class code |
| ASSUME documentation page "bidding agents" | <https://assume.readthedocs.io/en/stable/bidding_agents.html> | returned HTTP 404 |
| Cobweb: convergent when the supply curve is steeper than the demand curve in price-quantity terms, i.e. when demand is more elastic than supply at the equilibrium; divergent otherwise; adaptive expectations drive the fluctuations | Wikipedia, "Cobweb model" | opened through a summarising fetch; no stability condition for adaptive expectations is given |
| Nerlove (1958): adaptive expectations extend the parameter region where the cobweb is stable, shrink amplitudes, but can produce unstable or chaotic cycles | search summary (several hits, none opened) | **snippet only; unverified**; the runs in `spec-stability.md` 6.3 are the stb worker's own, not Nerlove's results |
| Backward-bending labour supply: income effect outweighs substitution effect at high wages; the page says nothing about stability or multiple equilibria | Wikipedia, "Backward bending supply curve of labour" | opened through a summarising fetch. The N-028 claim of multiple unstable equilibria stays **unverified** |
| Stylised-model results: the labour loop is stable iff demand elasticity > 1 (slope (1 - e)Q/p); two pinned outcomes for e < 1; stock-only belief rule oscillates 6% to 29%; evidence rule offset of 11% to 13% below the clearing price; hard cost floor freezes a market with a perishable; production cobweb stable iff es < ed under naive expectations | `spec-stability.md` section 6, Docker Python models | this spec's own stylised runs; not validated against data; placeholders |
| The Hawkins-Simon condition: all leading principal minors of `I - A` positive, equivalent to a non-negative `x` with `(I - A) x > 0`; the matrix is then productive | Wikipedia, "Hawkins-Simon condition" | opened via a summarising fetch |
| Leontief model: `x = (I - A)^-1 y`; productivity means the largest eigenvalue of `A` is below 1 | Wikipedia, "Input-output model" | opened via a summarising fetch |
| CGE models are calibrated so that the base data (a social accounting matrix) is an equilibrium of the model | Wikipedia, "Computable general equilibrium" | opened via a summarising fetch |
| Bland's rule never cycles, at the cost of speed | Wikipedia, "Bland's rule" | opened via a summarising fetch |
| Comparative advantage: specialise where opportunity cost is lowest; the terms of trade lie between the autarky ratios | Wikipedia, "Comparative advantage" | opened via a summarising fetch |
| Non-substitution theorem: with one primary factor and no joint production, price-to-wage ratios do not depend on the composition of final demand | Samuelson, Georgescu-Roegen (1951), via search results | **snippet only; unverified** |
| Factorio calculators solve recipe loops and byproducts with linear programming / a custom simplex | mods.factorio.com (Factory solver), forums.factorio.com (FactorioCalc) | **snippet only; unverified** |
| Sraffa (1960), von Neumann (1937), Leontief (1941) | not opened | named only; nothing set from them |
| The Wikipedia URLs for the non-substitution theorem and Ricardian trade theory returned 404 | | failed |
| Stone-Geary / linear expenditure system formulas | Wikipedia, "Stone-Geary utility function" | opened via a summarising fetch; no elasticities in the page. The claim that a good with `gamma = 0` has own-price elasticity -1 is the writer's hand derivation, **unverified** |
| Engel's law: the food share of spending falls as income rises; Engel's 1857 food income elasticity about 0.86 | Wikipedia, "Engel's law" | opened via a summarising fetch. No default set |
| CES: `sigma = 1/(1-rho)`, perfect substitutes and complements as limits | Wikipedia, "Constant elasticity of substitution" | opened via a summarising fetch |
| Victoria 3: staple and luxury goods, a prestige substitution weight of +50%, standard of living scales needs, a price band of 75% | Victoria 3 wiki, Goods and Pops pages | opened via a summarising fetch; the buy rule not read |
| Dwarf Fortress needs are not tiered by wealth or price and there is no money | Dwarf Fortress wiki, "Need" | opened via a summarising fetch |
| abcEconomics Household: only a Cobb-Douglas utility constructor is documented | abcEconomics, Household page | opened via a summarising fetch |
| USDA ERS demand-elasticity estimates (Seale, Regmi, Bernstein 2003) | search snippet; the summary PDF returned 404 | **snippet only; no numbers used** |
| AIDS, Working-Leser, Friedman, Godley-Lavoie consumption | not opened | named only |
| Replication of an agent-based model is not an explanation; joint replication of stylised facts; no single-seed bands | Windrum, Fagiolo and Moneta (2007), JASSS 10(2) 8, <https://www.jasss.org/10/2/8.html> | opened via a summarising fetch, not in full; used for the treatment of bands. No number |
| Gode and Sunder (1993) | search snippet | not opened; no number |
| Stylised-model results of the demand and spawner specs: a luxury labour loop converges only with schedule elasticity above 10 000 bp and about 10% budget headroom; the closed-economy ratio is 1.000 with upkeep only and 1.53 with wants; the entry share (not rotation, hysteresis or cooldown alone) removes the occupation swing: amplitude 0.29 at 200 bp and 0.00 at 50 bp | `spec-demand.md` 6.3 and 7.2, `spec-spawner.md` 16.1 | the workers' own Docker Python models; not validated against data; placeholders |
| No default in `spec-agents.md`, `spec-stability.md` or `spec-viability.md` is set from any source above | | all their numbers are placeholders |
| Logistic growth of a renewable stock, the maximum sustainable yield `rC/4`, the collapse of a stock under a constant quota above it, the catch-per-effort (Schaefer) form and the open-access equilibrium (Gordon) | general knowledge of the economics of renewable resources; not opened for `spec-resources.md` (section 21) | **unverified**; supply the structure (three shapes, the tipping point) and no number |
| A resource patch in a game that depletes, or loses yield to a floor as it empties (Factorio's finite and infinite patches are the example in mind) | general knowledge of games; the pages were not opened (`spec-resources.md` section 21) | **unverified**; only the idea that a yield may fall with depletion is used, as the abundance curve |
| No source on how fast news travelled, or how exact it was, in a pre-industrial economy was read; the defaults `RumourErrorBP` 500 and `RumourMaxAge` 60 are guesses | `spec-information.md` (handoff of the INF worker) | **unverified**; no source read, placeholders |
| Stylised-model results of RES: the regeneration arithmetic and the identity of an endowment hold over seeds; at half supply, first come in the rotation order left some of 1 000 weight-1 agents with nothing for a whole year, where the proportional ration gave each between 0.416 and 0.581 of its demand (one-tick jobs only, no strategy reacting to a denial, no money), and the ration is not fair in runs for a fixed mix of request sizes; the coupled stock-and-population loop agrees with the linearisation of `spec-resources.md` 8.3 (a constant stock neutral, a refill stock stable above a rate, a logistic stock stable only above the maximum-yield point and in a narrow basin) | `spec-resources.md` 5.3, 8.4, 17, `models/res/` | the worker's own Docker Python models; agreement with a linearisation, not validated against data and not a claim that any default is stable; placeholders |
| Stylised-model results of INF: the lag of a rumour adds to the transit delay of a shipper, so the stability limit of its gain is `2 sin(pi / (2 (2 (ell + k) + 1)))`; an independent information error is a weak damper, a shared error is worse than none, a spread of vintages lowers the limit and is rejected; a spread of the shippers' own thresholds of 0.8 times the freight settles the gap at 0.04 of it where an independent error of the same size leaves 0.6 and identical shippers 1.8 | `spec-information.md` 11, `models/inf/` | the worker's own Docker Python models; the shipper's rule is invented for them (MER's will decide), 50 shippers, single runs for the rows with an error; not validated against data; placeholders |
| Stylised-model results of EXT: the counterparty cycle (top-up, order, sweep) conserves money and each commodity over accounts and boundaries and leaves the counterparty accounts empty; a quote corridor of 45 to 75 confines a production cobweb that cycles between 20 and 120 without an outside, provided the capacity covers the swing, and the midpoint rule weakens the pull | `spec-external-sector.md` 6.7, 19, `models/ext/` | the worker's own Docker Python models; a uniform price stands in for the belief and clearing dynamics, so the shape of the effect is shown and not its size; not validated against data; placeholders |
| No default in `spec-resources.md`, `spec-information.md` or `spec-external-sector.md` is set from any source above | | all their numbers are placeholders |

Sources:

- Lengnick, M. (2013), "Agent-based macroeconomics: A baseline model",
  Journal of Economic Behavior and Organization 86, 102-120.
  <https://legacy.econ.tuwien.ac.at/lva/compeco.se/artikel/jebo_2013_agent_based_macroeconomics_a_baseline_model.pdf>
- Caiani, A., Godin, A., Caverzasi, E., Gallegati, M., Kinsella, S., Stiglitz,
  J. (2016), "Agent Based-Stock Flow Consistent Macroeconomics: Towards a
  Benchmark Model", Journal of Economic Dynamics and Control.
  <https://papers.ssrn.com/sol3/papers.cfm?abstract_id=2664125>
- Cliff, D., "ZIP60", zero-intelligence-plus traders.
  <https://eprints.soton.ac.uk/264078>
- "Production delays and price dynamics".
  <https://dare.uva.nl/id/076ce8c1-4387-462f-a172-48254cfd9cdc>
- Victoria 3 forum thread, "Economic Death Spiral".
  <https://steamcommunity.com/app/529340/discussions/0/4408543304949219350/>
- "Dwarf Fortress", Wikipedia. <https://en.wikipedia.org/wiki/Dwarf_Fortress>
- "Taking Malthus seriously", VoxEU.
  <https://cepr.org/voxeu/columns/taking-malthus-seriously>
- Clark, G. (2007), *A Farewell to Alms*, chapter 2, "The Logic of the Malthusian Economy".
  <https://faculty.econ.ucdavis.edu/faculty/gclark/Farewell%20to%20Alms/FTA-chapter2.pdf>
- Clark, G. (2006), draft of *A Farewell to Alms* (October 2006), chapters 4 and 5, draft pp. 90-93 and 100-103.
  <https://faculty.econ.ucdavis.edu/faculty/gclark/papers/FTA2006-1.pdf>
- "EVE Evolved: ISK sinks and faucets", Engadget (2010).
  <https://www.engadget.com/2010-10-24-eve-evolved-isk-sinks-and-faucets.html>
- abcEconomics documentation. <https://abce.readthedocs.io/en/master/>
- ASSUME documentation, "Custom unit example".
  <https://assume.readthedocs.io/en/stable/examples/03_custom_unit_example.html>
- Victoria 3 developer diary 9, "National Markets" (page did not load).
  <https://forum.paradoxplaza.com/forum/developer-diary/victoria-3-dev-diary-9-national-markets.1484917/>
- CCP, Monthly Economic Report, February 2026.
  <https://www.eveonline.com/news/view/monthly-economic-report-february-2026>
- Doran, J. and Parberry, I. (2012), "Emergent Economies for Role Playing Games", International Journal of Intelligent Games and Simulation 7(3), 35-48. <https://ianparberry.com/pubs/econ.pdf>
- "Cobweb model", Wikipedia. <https://en.wikipedia.org/wiki/Cobweb_model>
- Factorio, `RecipePrototype`. <https://lua-api.factorio.com/latest/prototypes/RecipePrototype.html>
- Victoria 3, "Production methods". <https://vic3.paradoxwikis.com/Production_methods>
- abcEconomics, Firm page. <https://abce.readthedocs.io/en/master/Firm.html>
- Lengnick (2013), household sections: see the Lengnick entry above.
- abcEconomics, Household and Trade pages. <https://abce.readthedocs.io/en/master/Household.html>, <https://abce.readthedocs.io/en/master/Trade.html>
- ASSUME, `base.py`. <https://raw.githubusercontent.com/assume-framework/assume/main/assume/common/base.py>
- Wikipedia: Hawkins-Simon condition, Input-output model, Computable general equilibrium, Bland's rule, Comparative advantage, Backward bending supply curve of labour.
- Factorio Factory solver. <https://mods.factorio.com/mod/factory_solver>; FactorioCalc thread. <https://forums.factorio.com/viewtopic.php?p=600437>
- Justman, M. and van der Beek, K. (2015), "Market forces shaping human capital in eighteenth-century London", Economic History Review 68(4). <https://tzin.bgu.ac.il/~kvdbeek/papers/Campbell%20EcHR_2015.pdf>
- Windrum, P., Fagiolo, G. and Moneta, A. (2007), "Empirical validation of agent-based models: alternatives and prospects", JASSS 10(2) 8. <https://www.jasss.org/10/2/8.html>
