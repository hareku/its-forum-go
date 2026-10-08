// Package rc016 encodes ITS FORUM RC-016 version 1.0 bicycle, pedestrian,
// ordinary roadside, and CSMA roadside messages. The explicit family decoders
// avoid guessing experimental identifiers. RC-013 frames are independently
// available in the github.com/hareku/its-forum-go/rc013v1 module.
// There is no device communication, encryption,
// or object fusion processing.
//
// Integer fields preserve wire representations, including unknown values and
// unavailable sentinels. Decode copies retained bytes; MarshalBinary returns
// fresh bytes. Public buffers and pointers are editable, and ordinary Go struct
// copies share those buffers and pointers. Validate reports semantic concerns
// separately from mandatory structural checks.
//
// Experiment-specific application IDs and sensor/extension layouts require
// explicit profiles, copied at construction or decode. Unknown payloads remain
// opaque; edits moving unresolved data return ErrLayout. New constructors are
// the explicit placement boundary. Header lengths and counts are derived from
// the model rather than stale decoded observations.
//
// Ordinary roadside headers follow table 4-3's four-bit version and sixteen-byte
// total, despite the contradictory one-bit prose in section 4.4.2.1.1(3).
// Chapter-3 common frames retain the fixed delay bits at every personal level;
// appendix-2 nonstorage wording does not define another packing. Bicycle cadence
// stays raw because its definition and resolution cell disagree about units.
package rc016
