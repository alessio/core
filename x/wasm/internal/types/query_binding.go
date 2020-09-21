package types

import (
	"encoding/json"

	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// WasmQuerierInterface - query registration interface for other modules
type WasmQuerierInterface interface {
	Query(ctx sdk.Context, request wasmTypes.QueryRequest) ([]byte, error)
	QueryCustom(ctx sdk.Context, data json.RawMessage) ([]byte, error)
}

// Querier - wasm query handler
type Querier struct {
	WasmContext WasmContext
	Queriers    map[string]WasmQuerierInterface
}

// NewModuleQuerier return wasm querier
func NewModuleQuerier() Querier {
	return Querier{
		Queriers: make(map[string]WasmQuerierInterface),
	}
}

// WasmCustomQuery - wasm custom query
type WasmCustomQuery struct {
	Route     string          `json:"route"`
	QueryData json.RawMessage `json:"query_data"`
}

var _ wasmTypes.Querier = Querier{}

// Routes of pre-determined wasm querier
const (
	WasmQueryRouteBank     = "bank"
	WasmQueryRouteStaking  = "staking"
	WasmQueryRouteMarket   = "market"
	WasmQueryRouteTreasury = "treasury"
	WasmQueryRouteWasm     = "wasm"
)

// WithCtx returns new querier with context
func (q Querier) WithCtx(ctx WasmContext) Querier {
	q.WasmContext = ctx
	return q
}

// GasConsumed consume gas in the current context
func (q Querier) GasConsumed() uint64 {
	return q.WasmContext.Ctx.GasMeter().GasConsumed()
}

// Query - interface for wasmTypes.Querier
func (q Querier) Query(request wasmTypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	// set a limit for a ctx
	// gasLimit passed from the go-cosmwasm part, so need to divide it with gas multiplier
	ctx := q.WasmContext.Ctx.WithGasMeter(sdk.NewGasMeter(gasLimit / q.WasmContext.GasMultiplier))

	// make sure we charge the higher level context even on panic
	defer func() {
		q.WasmContext.Ctx.GasMeter().ConsumeGas(ctx.GasMeter().GasConsumed(), "contract sub-query")
	}()

	// do the query

	switch {
	case request.Bank != nil:
		if querier, ok := q.Queriers[WasmQueryRouteBank]; ok {
			return querier.Query(ctx, request)
		}

		return nil, sdkerrors.Wrap(ErrNoRegisteredQuerier, WasmQueryRouteBank)
	case request.Custom != nil:
		var customQuery WasmCustomQuery
		err := json.Unmarshal(request.Custom, &customQuery)
		if err != nil {
			return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
		}

		if querier, ok := q.Queriers[customQuery.Route]; ok {
			return querier.QueryCustom(ctx, customQuery.QueryData)
		}

		return nil, sdkerrors.Wrap(ErrNoRegisteredQuerier, customQuery.Route)
	case request.Staking != nil:
		if querier, ok := q.Queriers[WasmQueryRouteStaking]; ok {
			return querier.Query(ctx, request)
		}

		return nil, sdkerrors.Wrap(ErrNoRegisteredQuerier, WasmQueryRouteStaking)
	case request.Wasm != nil:
		if querier, ok := q.Queriers[WasmQueryRouteWasm]; ok {
			return querier.Query(ctx, request)
		}

		return nil, sdkerrors.Wrap(ErrNoRegisteredQuerier, WasmQueryRouteWasm)
	}

	return nil, wasmTypes.Unknown{}
}
