/*
FILE: internal/ws/protocol.go

DESCRIPTION:
On-the-wire types for Bybit V5 WebSocket protocol. Kept in a separate file
because the protocol is small and mostly schema (no logic). All structs
here must mirror the documented Bybit V5 envelope exactly so json-iterator
can decode incoming frames in one pass.

OUTBOUND OPS we issue:

  {"op":"auth",     "args":[apiKey, expires, signature]}
  {"op":"subscribe","args":["topic1","topic2",...]}
  {"op":"unsubscribe","args":["topic"]}
  {"op":"ping"}

INBOUND ENVELOPES we observe:

  Acks (auth/subscribe/pong):
    {"op":"auth","success":true,"ret_msg":"...","conn_id":"..."}
    {"op":"subscribe","success":true,"ret_msg":"","conn_id":"...","req_id":"..."}
    {"op":"pong","args":["<ts>"]}

  Public/private push (typed by topic):
    {"topic":"orderbook.50.BTCUSDT","ts":1700000000000,"type":"snapshot","data":{...}}
    {"topic":"position","creationTime":1700,"data":[...]}

The Envelope below is the unified shape: ops have op/success/ret_msg, push
frames have topic/data. The dispatcher distinguishes by which fields are
populated.
*/

package ws

import "github.com/tonymontanov/go-bybit/v2/internal/codec"

// OutboundOp is the JSON payload for a control frame we send to Bybit.
type OutboundOp struct {
	Op    string `json:"op"`
	Args  []any  `json:"args,omitempty"`
	ReqID string `json:"req_id,omitempty"`
}

// Envelope captures the union of inbound frames.
//
//   - For control acks (auth/subscribe/unsubscribe/pong): Op, Success,
//     RetMsg, ConnID, ReqID, Args are populated; Topic is empty.
//   - For data pushes:                                   Topic and Data
//     are populated; Op is empty (or "pong" when Bybit echoes pong via
//     this channel — handled by the dispatcher).
type Envelope struct {
	// Control fields.
	Op      string `json:"op"`
	Success *bool  `json:"success,omitempty"`
	RetMsg  string `json:"ret_msg"`
	ConnID  string `json:"conn_id"`
	ReqID   string `json:"req_id"`
	Args    []any  `json:"args,omitempty"`
	// Data fields.
	Topic        string        `json:"topic"`
	Type         string        `json:"type"`
	TsMs         int64         `json:"ts"`
	CreationTime int64         `json:"creationTime"`
	Data         codec.RawJSON `json:"data"`
}

// IsControl returns true when the envelope describes a control-channel
// reply (op != "" and topic == "").
func (e *Envelope) IsControl() bool {
	return e.Op != "" && e.Topic == ""
}

// IsPush returns true when the envelope describes a data push (topic != "").
func (e *Envelope) IsPush() bool {
	return e.Topic != ""
}
