# besøg backend

## how to run

podman build -t dai-api .
podman run -p 8080:8080 --env-file .env -v ./data.db:/app/data.db localhost/dai-api:latest

## vision

En hjemmeside til konsulent rapport

rapport from nem og klikke
vedhæft billeder
lokation

integration til AdvoPro, hvilket betyder upload pdf

lang tid bliver brugt til at pakke en rapport.

personlig statestik
hvor mange biler har man fået ind
hvor mange besøg har man lavet.
en smule gamification

# TODO

use cycle

konsulent besøger hjemmeside
logger ind
kigger på sin dag og besøg
klikker på den første og åbner en questionaire som er adaptiv
trykker send, hvor både data og tidspunkt og person-placering bliver sendt med,
trykker videre på næste sag og gentager

admin besøger hjemmesiden
placerer besøg som skal findes enten igennem excel eller noget andet i fremtiden
tjekker på besøgs historik på alle konsulenter
trækker besøgs data ud og skriver ind i advopro (måske automatisk i fremtiden)

## done

login
register
besøgs form
Hvis konsulenten har tjekket at bilen er skadet så skal han også vedhæfte et billede

# Route Segment / Boundary Design

## Problem

Run TSP per-island (or per-region) while keeping stops for each island
contiguous, and let users pin specific stops (e.g. forced start point,
forced last stop of the day) so the optimizer never moves them.

## Decision

- `SegmentIndex uint` groups visits into contiguous route chunks.
  No extra table, no bool flags for "locked" stops.
- `Stopnr` stays the single global order the GUI reads/writes. Its meaning
  does not change.
- Segment membership + `Stopnr` order together imply everything else:

### Contiguity

All rows sharing a `SegmentIndex` form one uninterrupted block of `Stopnr`.

### Boundaries (first/last per segment)

Derived, not stored. One pass over stops sorted by `Stopnr`:

```go
func BoundaryVisitIDs(visits []Visit) map[uint]string {
    sort.Slice(visits, func(i, j int) bool { return visits[i].Stopnr < visits[j].Stopnr })
    boundaries := map[uint]string{}
    for i, v := range visits {
        if i == 0 || visits[i-1].SegmentIndex != v.SegmentIndex {
            boundaries[v.ID] = "first"
        }
        if i == len(visits)-1 || visits[i+1].SegmentIndex != v.SegmentIndex {
            if boundaries[v.ID] == "first" {
                boundaries[v.ID] = "first+last" // single-visit segment
            } else {
                boundaries[v.ID] = "last"
            }
        }
    }
    return boundaries
}
```

Nothing to null out when segments split or merge — boundaries recompute
automatically from whatever `SegmentIndex` values exist.

### Locking a single stop (forced last stop of the day, etc.)

Give it its own `SegmentIndex` of size 1. The boundary detector marks it
`"first+last"` automatically, and a segment of 1 has nothing to reorder —
the optimizer skips it outright.

To make a stop optional again (free to be optimized), just move it back
into the neighboring segment's `SegmentIndex`. No flag to clear.

## Example

```
visitid | segmentIndex | stopnr
1       | 0            | 1
2       | 0            | 2
4       | 0            | 3
6       | 1            | 4
8       | 1            | 5
10      | 1            | 6
3       | 2            | 7
5       | 2            | 8
7       | 2            | 9
9       | 3            | 10   <- own segment, size 1, locked as final stop
```

## Optimizer contract

- Per `SegmentIndex`, run TSP only over interior stops (exclude first/last,
  or use fixed-endpoint TSP if boundaries should anchor route geometry).
- Segments of size 1 are skipped entirely.
- After each segment's local order is computed:
  `Stopnr = cumulative offset + local order`.

## Rejected alternative

`FirstStop *bool` / `LastStop *bool` on `Visit`. Allows invalid states
(two "first"s in one segment, stale flags after merge/split) and requires
manual nulling whenever segments change. Deriving from `SegmentIndex` +
`Stopnr` self-heals on any reshuffle and can't represent an invalid state.

## currently

### Fix deployment

podman
automated deployment

### Add functionanility
