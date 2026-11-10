package otelobserver

// kind is the OpenTelemetry instrument kind of a row of the catalogue.
type kind uint8

const (
	kCounter kind = iota
	kUpDown
	kGauge
	kHistogram
)

// spec is one row of the catalogue (spec-observability.md 6.2): the name, kind,
// unit and description of an instrument and the keys of its attributes, in the
// order a series is keyed by them.
type spec struct {
	name  string
	kind  kind
	unit  string
	desc  string
	attrs []string
}

// The instruments, in the order of the tables of spec-observability.md 6.2.1
// and 6.2.2. The index of an instrument is its constant.
const (
	iWorldStepDuration = iota
	iWorldStepFailed
	iMarketTrades
	iMarketVolume
	iEstateSold
	iMarketClearingPrice
	iMarketPriceAge
	iMarketUnmatched
	iMarketOrders
	iMarketOrdersRejected
	iBeliefCentre
	iBeliefAtBound
	iBeliefOutcomes
	iBeliefLimited
	iRecipeExecutions
	iRecipeRefused
	iRecipeAbandoned
	iRecipeOverflow
	iProductionCostAnchor
	iProductionHalted
	iWorkDeclined
	iAgentCount
	iAgentRemovals
	iHouseholdCount
	iHouseholdUnfed
	iUpkeepShort
	iHouseholdExits
	iRoleChanges
	iEntryLimited
	iHouseholdEntries
	iWantUnfunded
	iHouseholdRelocations
	iMoneyFlow
	iCommodityFlow
	iCommodityWear
	iMoneyFlowUnapplied
	iHouseholdBirths
	iHouseholdDeaths
	iTransportShipments
	iTransportInTransit
	iTransportCargoSpoiled
	iEndowmentDrawn
	iEndowmentRegenerated
	iEndowmentShort
	iEndowmentStock
	iInformationRumours
	iExternalUnits
	iExternalValue
	iExternalPrice
	iHouseholdOwnUse
	iOwnUseUnits
	iMigrationStarted
	iMigrationLimited
	iMigrationOnJourney
	iContractFormed
	iContractValue
	iContractEscrow
	iMerchantDecisions
	iProductionPending
	iRecipeInProgress
	iMarketStock
	iEstateAtFloor
	iLedgerLots
	iMoneySupply
	iWelfareRatio
	iWelfareRatioHouseholds
	iWelfareFloorHeld
	iWelfareClamped
	iWageObserved
	iHouseholdPrivationMax
	iRoleExpectedRatio
	iRoleViable
	iRoleAgents
	iWantChoked
	iMoneyFlowLimited
	iPopulationLimited
	iTransportCargoDiverted
	iTransportRejected
	iInformationReaders
	iExternalCut
	iExternalUnfilled
	iExternalClamped
	iOwnUseBlocked
	iHouseholdOwnUseChanges
	iMigrationUnsettled
	iContractSettled
	iContractUnformed
	iContractRefused
	iPopulationNaturalIncrease
	iPopulationCap
	iPopulationReplacementRatio
	iTransportOverdue
	iTransportAway
	iMigrationPipeline
	nInstruments
)

// catalogue is the one table every instrument is built from (New). The Stage 3
// rows are appended by the module plans that own them.
var catalogue = [nInstruments]spec{
	iWorldStepDuration:          {name: "economysim.world.step.duration", kind: kHistogram, unit: "s", desc: "Wall-clock duration of one completed step, measured by the adapter.", attrs: []string{}},
	iWorldStepFailed:            {name: "economysim.world.step.failed", kind: kCounter, unit: "{step}", desc: "Steps that ended with the world failed.", attrs: []string{}},
	iMarketTrades:               {name: "economysim.market.trades", kind: kCounter, unit: "{trade}", desc: "Trades executed.", attrs: []string{keyCommodity}},
	iMarketVolume:               {name: "economysim.market.volume", kind: kCounter, unit: "{unit}", desc: "Units traded.", attrs: []string{keyCommodity}},
	iEstateSold:                 {name: "economysim.estate.sold", kind: kCounter, unit: "{unit}", desc: "Units sold by an estate, the forced seller of a removed agent.", attrs: []string{keyCommodity, keyRegion}},
	iMarketClearingPrice:        {name: "economysim.market.clearing_price", kind: kGauge, unit: "{minor_unit}", desc: "Published price of the last clearing of a market.", attrs: []string{keyCommodity, keyRegion}},
	iMarketPriceAge:             {name: "economysim.market.price_age", kind: kGauge, unit: "{tick}", desc: "Ticks since the last trade set the price of a market.", attrs: []string{keyCommodity, keyRegion}},
	iMarketUnmatched:            {name: "economysim.market.unmatched", kind: kCounter, unit: "{unit}", desc: "Units left unmatched at clearing, by side.", attrs: []string{keyCommodity, keySide}},
	iMarketOrders:               {name: "economysim.market.orders", kind: kCounter, unit: "{order}", desc: "Orders placed.", attrs: []string{keyCommodity, keyRegion, keySide}},
	iMarketOrdersRejected:       {name: "economysim.market.orders.rejected", kind: kCounter, unit: "{order}", desc: "Orders rejected, by reason.", attrs: []string{keyCommodity, keyReason}},
	iBeliefCentre:               {name: "economysim.belief.centre", kind: kGauge, unit: "{minor_unit}", desc: "Mean centre of the price beliefs of a market.", attrs: []string{keyCommodity, keyRegion}},
	iBeliefAtBound:              {name: "economysim.belief.at_bound", kind: kGauge, unit: "{basis_point}", desc: "Share of order outcomes whose belief centre ended on a bound, in basis points.", attrs: []string{keyCommodity, keyRegion, keyBound}},
	iBeliefOutcomes:             {name: "economysim.belief.outcomes", kind: kCounter, unit: "{outcome}", desc: "Order outcomes processed by the belief update.", attrs: []string{keyCommodity, keyRegion}},
	iBeliefLimited:              {name: "economysim.belief.limited", kind: kCounter, unit: "{outcome}", desc: "Belief updates cut by a conflict or a clamp.", attrs: []string{keyCommodity, keyRegion, keyBeliefLimit}},
	iRecipeExecutions:           {name: "economysim.recipe.executions", kind: kCounter, unit: "{execution}", desc: "Recipe executions completed.", attrs: []string{keyRecipe}},
	iRecipeRefused:              {name: "economysim.recipe.refused", kind: kCounter, unit: "{household}", desc: "Households whose job request was refused, by reason.", attrs: []string{keyRecipe, keyReason}},
	iRecipeAbandoned:            {name: "economysim.recipe.abandoned", kind: kCounter, unit: "{execution}", desc: "Recipe executions ended without output, by reason.", attrs: []string{keyRecipe, keyReason}},
	iRecipeOverflow:             {name: "economysim.recipe.overflow", kind: kCounter, unit: "{unit}", desc: "Units of output disposed that are not a discard output.", attrs: []string{keyCommodity}},
	iProductionCostAnchor:       {name: "economysim.production.cost_anchor", kind: kGauge, unit: "{minor_unit}", desc: "Cost anchor in force for a commodity in a region; 0 is unknown.", attrs: []string{keyCommodity, keyRegion}},
	iProductionHalted:           {name: "economysim.production.halted", kind: kCounter, unit: "{household}", desc: "Households for which the production halt skipped a recipe.", attrs: []string{keyRole, keyRecipe}},
	iWorkDeclined:               {name: "economysim.work.declined", kind: kCounter, unit: "{household}", desc: "Free households that requested no job, by decline.", attrs: []string{keyRole, keyWorkDecline}},
	iAgentCount:                 {name: "economysim.agent.count", kind: kUpDown, unit: "{agent}", desc: "Live agents, by role.", attrs: []string{keyRole}},
	iAgentRemovals:              {name: "economysim.agent.removals", kind: kCounter, unit: "{agent}", desc: "Agents removed entirely, by cause.", attrs: []string{keyRole, keyCause}},
	iHouseholdCount:             {name: "economysim.household.count", kind: kUpDown, unit: "{household}", desc: "Households, the sum of agent weights, by role.", attrs: []string{keyRole}},
	iHouseholdUnfed:             {name: "economysim.household.unfed", kind: kUpDown, unit: "{household}", desc: "Households that went unfed at the last consumption phase, by role.", attrs: []string{keyRole}},
	iUpkeepShort:                {name: "economysim.upkeep.short", kind: kCounter, unit: "{unit}", desc: "Upkeep units that households needed and did not hold.", attrs: []string{keyCommodity, keyRole}},
	iHouseholdExits:             {name: "economysim.household.exits", kind: kCounter, unit: "{household}", desc: "Households that left by the failure ladder or a consumer removal.", attrs: []string{keyRole, keyCause}},
	iRoleChanges:                {name: "economysim.role.changes", kind: kCounter, unit: "{change}", desc: "Role changes, by destination role.", attrs: []string{keyRole}},
	iEntryLimited:               {name: "economysim.entry.limited", kind: kCounter, unit: "{case}", desc: "Cases in which a cap, cooldown or refusal limited entry or exit.", attrs: []string{keyEntryLimit}},
	iHouseholdEntries:           {name: "economysim.household.entries", kind: kCounter, unit: "{household}", desc: "Households that entered a region.", attrs: []string{keyRegion}},
	iWantUnfunded:               {name: "economysim.want.unfunded", kind: kCounter, unit: "{unit}", desc: "Units of a want planned and not funded.", attrs: []string{keyWant, keyRole}},
	iHouseholdRelocations:       {name: "economysim.household.relocations", kind: kCounter, unit: "{household}", desc: "Households relocated between regions.", attrs: []string{keyRegionFrom, keyRegionTo, keyCause}},
	iMoneyFlow:                  {name: "economysim.money.flow", kind: kCounter, unit: "{minor_unit}", desc: "Money moved across the world boundary, by ledger reason and kind.", attrs: []string{keyReason, keyKind}},
	iCommodityFlow:              {name: "economysim.commodity.flow", kind: kCounter, unit: "{unit}", desc: "Goods moved across the world boundary, by ledger reason and kind.", attrs: []string{keyReason, keyKind, keyCommodity}},
	iCommodityWear:              {name: "economysim.commodity.wear", kind: kCounter, unit: "{unit}", desc: "Durability units removed from items by wear.", attrs: []string{keyCommodity}},
	iMoneyFlowUnapplied:         {name: "economysim.money.flow.unapplied", kind: kCounter, unit: "{minor_unit}", desc: "Money a flow computed and could not apply.", attrs: []string{keyReason}},
	iHouseholdBirths:            {name: "economysim.household.births", kind: kCounter, unit: "{household}", desc: "Households formed by the population module.", attrs: []string{keyRole}},
	iHouseholdDeaths:            {name: "economysim.household.deaths", kind: kCounter, unit: "{household}", desc: "Households dissolved by the population module.", attrs: []string{keyRole}},
	iTransportShipments:         {name: "economysim.transport.shipments", kind: kCounter, unit: "{shipment}", desc: "Shipments dispatched, by transport mode.", attrs: []string{keyMode}},
	iTransportInTransit:         {name: "economysim.transport.in_transit", kind: kUpDown, unit: "{unit}", desc: "Units on shipments in flight.", attrs: []string{keyCommodity}},
	iTransportCargoSpoiled:      {name: "economysim.transport.cargo.spoiled", kind: kCounter, unit: "{unit}", desc: "Units that spoiled in transit.", attrs: []string{keyCommodity}},
	iEndowmentDrawn:             {name: "economysim.endowment.drawn", kind: kCounter, unit: "{unit}", desc: "Units drawn from an endowment.", attrs: []string{keyResource, keyRegion}},
	iEndowmentRegenerated:       {name: "economysim.endowment.regenerated", kind: kCounter, unit: "{unit}", desc: "Units an endowment regenerated.", attrs: []string{keyResource, keyRegion}},
	iEndowmentShort:             {name: "economysim.endowment.short", kind: kCounter, unit: "{unit}", desc: "Units asked of an endowment and not granted.", attrs: []string{keyResource, keyRegion}},
	iEndowmentStock:             {name: "economysim.endowment.stock", kind: kGauge, unit: "{unit}", desc: "Stock of an endowment after the last tick.", attrs: []string{keyResource, keyRegion}},
	iInformationRumours:         {name: "economysim.information.rumours", kind: kCounter, unit: "{rumour}", desc: "Rumours offered to informed agents, by outcome and reader region.", attrs: []string{keyOutcome, keyRegion}},
	iExternalUnits:              {name: "economysim.external.units", kind: kCounter, unit: "{unit}", desc: "Units traded with an outside node.", attrs: []string{keyNode, keyCommodity, keySide}},
	iExternalValue:              {name: "economysim.external.value", kind: kCounter, unit: "{minor_unit}", desc: "Money paid or received in trades with an outside node.", attrs: []string{keyNode, keyCommodity, keySide}},
	iExternalPrice:              {name: "economysim.external.price", kind: kGauge, unit: "{minor_unit}", desc: "Limit price of an outside line.", attrs: []string{keyNode, keyRegion, keyCommodity, keySide}},
	iHouseholdOwnUse:            {name: "economysim.household.own_use", kind: kGauge, unit: "{household}", desc: "Households in the own-use mode.", attrs: []string{keyRole, keyRegion}},
	iOwnUseUnits:                {name: "economysim.own_use.units", kind: kCounter, unit: "{unit}", desc: "Units created by own-use jobs.", attrs: []string{keyCommodity}},
	iMigrationStarted:           {name: "economysim.migration.started", kind: kCounter, unit: "{household}", desc: "Households that started a migration journey.", attrs: []string{keyRegionFrom, keyRegionTo}},
	iMigrationLimited:           {name: "economysim.migration.limited", kind: kCounter, unit: "{household}", desc: "Migration review cases that did not start a journey, by outcome and region.", attrs: []string{keyLimit, keyRegion}},
	iMigrationOnJourney:         {name: "economysim.migration.on_journey", kind: kUpDown, unit: "{household}", desc: "Households on a migration journey.", attrs: []string{}},
	iContractFormed:             {name: "economysim.contract.formed", kind: kCounter, unit: "{contract}", desc: "Delivery contracts formed.", attrs: []string{keyCommodity}},
	iContractValue:              {name: "economysim.contract.value", kind: kCounter, unit: "{minor_unit}", desc: "Money escrowed by formed delivery contracts.", attrs: []string{keyCommodity}},
	iContractEscrow:             {name: "economysim.contract.escrow", kind: kUpDown, unit: "{minor_unit}", desc: "Money held in escrow by open delivery contracts.", attrs: []string{}},
	iMerchantDecisions:          {name: "economysim.merchant.decisions", kind: kCounter, unit: "{lane}", desc: "Lane decisions of merchants.", attrs: []string{keyDecision}},
	iProductionPending:          {name: "economysim.production.pending", kind: kGauge, unit: "{unit}", desc: "Output of jobs in progress, held by no account.", attrs: []string{keyCommodity, keyRegion}},
	iRecipeInProgress:           {name: "economysim.recipe.in_progress", kind: kGauge, unit: "{execution}", desc: "Executions of jobs in progress.", attrs: []string{keyRecipe, keyRegion}},
	iMarketStock:                {name: "economysim.market.stock", kind: kGauge, unit: "{unit}", desc: "Units held in a market, by holder.", attrs: []string{keyCommodity, keyRegion, keyHolder}},
	iEstateAtFloor:              {name: "economysim.estate.at_floor", kind: kGauge, unit: "{unit}", desc: "Units of estate listings priced at the resolution floor.", attrs: []string{keyCommodity, keyRegion}},
	iLedgerLots:                 {name: "economysim.ledger.lots", kind: kGauge, unit: "{lot}", desc: "Lots held in the ledger accounts of a market.", attrs: []string{keyCommodity, keyRegion}},
	iMoneySupply:                {name: "economysim.money.supply", kind: kGauge, unit: "{minor_unit}", desc: "Money held by a sector of a region.", attrs: []string{keyRegion, keySector}},
	iWelfareRatio:               {name: "economysim.welfare.ratio", kind: kGauge, unit: "{basis_point}", desc: "Mean welfare ratio of a role in a region, in basis points.", attrs: []string{keyRole, keyRegion}},
	iWelfareRatioHouseholds:     {name: "economysim.welfare.ratio.households", kind: kGauge, unit: "{household}", desc: "Households behind the mean welfare ratio.", attrs: []string{keyRole, keyRegion}},
	iWelfareFloorHeld:           {name: "economysim.welfare.floor_held", kind: kGauge, unit: "{basis_point}", desc: "Share of households held to work by the work floor, in basis points.", attrs: []string{keyRole, keyRegion}},
	iWelfareClamped:             {name: "economysim.welfare.clamped", kind: kGauge, unit: "{household}", desc: "Households whose welfare record holds a field at the clamp.", attrs: []string{keyRole, keyRegion}},
	iWageObserved:               {name: "economysim.wage.observed", kind: kGauge, unit: "{minor_unit}", desc: "Observed wage of a role in a region.", attrs: []string{keyRole, keyRegion}},
	iHouseholdPrivationMax:      {name: "economysim.household.privation.max", kind: kGauge, unit: "{tick}", desc: "Largest privation of an agent of a role, in ticks.", attrs: []string{keyRole, keyRegion}},
	iRoleExpectedRatio:          {name: "economysim.role.expected_ratio", kind: kGauge, unit: "{basis_point}", desc: "Expected ratio of a role that ranks it for entry and switching.", attrs: []string{keyRole, keyRegion}},
	iRoleViable:                 {name: "economysim.role.viable", kind: kGauge, unit: "{flag}", desc: "1 if a role is viable at the entry ratio, else 0.", attrs: []string{keyRole, keyRegion}},
	iRoleAgents:                 {name: "economysim.role.agents", kind: kGauge, unit: "{agent}", desc: "Agents of a role in a region.", attrs: []string{keyRole, keyRegion}},
	iWantChoked:                 {name: "economysim.want.choked", kind: kCounter, unit: "{household}", desc: "Households whose price ratio for a want was at the choke price.", attrs: []string{keyWant, keyRole}},
	iMoneyFlowLimited:           {name: "economysim.money.flow.limited", kind: kCounter, unit: "{household}", desc: "Households a money flow could not serve in full, by limit.", attrs: []string{keyReason, keyFlowLimit}},
	iPopulationLimited:          {name: "economysim.population.limited", kind: kCounter, unit: "{household}", desc: "Births and deaths that fell short of the schedule, by limit.", attrs: []string{keyPopulationLimit}},
	iTransportCargoDiverted:     {name: "economysim.transport.cargo.diverted", kind: kCounter, unit: "{unit}", desc: "Units that reached the destination and went to its estate, not the consignee.", attrs: []string{keyCommodity}},
	iTransportRejected:          {name: "economysim.transport.rejected", kind: kCounter, unit: "{request}", desc: "Shipment requests refused, by mode and reason.", attrs: []string{keyMode, keyReason}},
	iInformationReaders:         {name: "economysim.information.readers", kind: kCounter, unit: "{agent}", desc: "Informed agents given a view of rumours.", attrs: []string{keyRegion}},
	iExternalCut:                {name: "economysim.external.cut", kind: kCounter, unit: "{unit}", desc: "Units an outside line wanted and did not offer, by cut.", attrs: []string{keyNode, keyCommodity, keySide, keyExternalCut}},
	iExternalUnfilled:           {name: "economysim.external.unfilled", kind: kCounter, unit: "{unit}", desc: "Units an outside line offered at its quote and the world did not take.", attrs: []string{keyNode, keyCommodity, keySide}},
	iExternalClamped:            {name: "economysim.external.clamped", kind: kCounter, unit: "{case}", desc: "Outside quotes moved by a clamp of the factor or the price.", attrs: []string{keyNode, keyCommodity}},
	iOwnUseBlocked:              {name: "economysim.own_use.blocked", kind: kCounter, unit: "{household}", desc: "Households in the own-use mode that were free and started nothing.", attrs: []string{keyRole}},
	iHouseholdOwnUseChanges:     {name: "economysim.household.own_use.changes", kind: kCounter, unit: "{household}", desc: "Households that entered or left the own-use mode.", attrs: []string{keyRole, keyOwnUseChange}},
	iMigrationUnsettled:         {name: "economysim.migration.unsettled", kind: kCounter, unit: "{household}", desc: "Households the settling rule kept from being reviewed for migration.", attrs: []string{}},
	iContractSettled:            {name: "economysim.contract.settled", kind: kCounter, unit: "{contract}", desc: "Delivery contracts ended, by cause.", attrs: []string{keyCause}},
	iContractUnformed:           {name: "economysim.contract.unformed", kind: kCounter, unit: "{match}", desc: "Contract matches that failed to form, by failure.", attrs: []string{keyContractFailure}},
	iContractRefused:            {name: "economysim.contract.refused", kind: kCounter, unit: "{request}", desc: "Delivery offers and takes refused at admission.", attrs: []string{keyContractRequest}},
	iPopulationNaturalIncrease:  {name: "economysim.population.natural_increase", kind: kGauge, unit: "{basis_point}", desc: "Mean birth rate less death rate of a role, in basis points a year.", attrs: []string{keyRole, keyRegion}},
	iPopulationCap:              {name: "economysim.population.cap", kind: kGauge, unit: "{household}", desc: "Households a region may hold.", attrs: []string{keyRegion}},
	iPopulationReplacementRatio: {name: "economysim.population.replacement_ratio", kind: kGauge, unit: "{basis_point}", desc: "Welfare ratio at which births equal deaths.", attrs: []string{}},
	iTransportOverdue:           {name: "economysim.transport.overdue", kind: kGauge, unit: "{shipment}", desc: "Shipments in flight at or past their arrival tick, by mode.", attrs: []string{keyMode}},
	iTransportAway:              {name: "economysim.transport.away", kind: kGauge, unit: "{agent}", desc: "Agents away on a self-carried trip.", attrs: []string{}},
	iMigrationPipeline:          {name: "economysim.migration.pipeline", kind: kGauge, unit: "{household}", desc: "Households on a journey out of and into a region.", attrs: []string{keyRegion, keyDirection}},
}

// The attribute keys of the catalogue (rules 5.4.1). A key outside the
// namespace is a defect.
const (
	keyActivity        = "economysim.activity"
	keyActor           = "economysim.actor"
	keyBar             = "economysim.bar"
	keyBeliefLimit     = "economysim.belief_limit"
	keyBody            = "economysim.body"
	keyBound           = "economysim.bound"
	keyCause           = "economysim.cause"
	keyClamp           = "economysim.clamp"
	keyCommodity       = "economysim.commodity"
	keyControl         = "economysim.control"
	keyCoverKind       = "economysim.cover_kind"
	keyCoverLimit      = "economysim.cover_limit"
	keyCoverLoss       = "economysim.cover_loss"
	keyCreditLimit     = "economysim.credit_limit"
	keyDecision        = "economysim.decision"
	keyDimension       = "economysim.dimension"
	keyEntryLimit      = "economysim.entry_limit"
	keyEntryOutcome    = "economysim.entry_outcome"
	keyFacilityClass   = "economysim.facility_class"
	keyFacilityLimit   = "economysim.facility_limit"
	keyField           = "economysim.field"
	keyHolder          = "economysim.holder"
	keyKind            = "economysim.kind"
	keyLimit           = "economysim.limit"
	keyMode            = "economysim.mode"
	keyNode            = "economysim.node"
	keyOutcome         = "economysim.outcome"
	keyPolicyCut       = "economysim.policy_cut"
	keyPolicyHold      = "economysim.policy_hold"
	keyProject         = "economysim.project"
	keyReason          = "economysim.reason"
	keyRecipe          = "economysim.recipe"
	keyRegion          = "economysim.region"
	keyRegionFrom      = "economysim.region.from"
	keyRegionTo        = "economysim.region.to"
	keyResource        = "economysim.resource"
	keyRight           = "economysim.right"
	keyRiskKind        = "economysim.risk_kind"
	keyRole            = "economysim.role"
	keyRoomLimit       = "economysim.room_limit"
	keySector          = "economysim.sector"
	keySide            = "economysim.side"
	keyStoreClass      = "economysim.store_class"
	keyTier            = "economysim.tier"
	keyWant            = "economysim.want"
	keyWorkDecline     = "economysim.work_decline"
	keyContractFailure = "economysim.contract_failure"
	keyContractRequest = "economysim.contract_request"
	keyDirection       = "economysim.direction"
	keyExternalCut     = "economysim.external_cut"
	keyFlowLimit       = "economysim.flow_limit"
	keyOwnUseChange    = "economysim.own_use_change"
	keyPopulationLimit = "economysim.population_limit"
)
