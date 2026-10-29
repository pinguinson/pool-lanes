# Pool Lanes

Tracks how booked-up lane swimming is at London Aquatics Centre, so you can tell how busy a pool will be at a given time before going.

## Language

### Places

**Centre**:
A leisure centre run by Everyone Active. Only London Aquatics Centre is tracked.
_Avoid_: Venue, facility, club

**Pool**:
One of the Centre's lap-swimming pools: the Competition Pool or the Training Pool. The Diving Pool isn't tracked.
_Avoid_: "the pool" without saying which

### Sessions

**Swim Session**:
A bookable time slot for swimming in one or more Lanes of a Pool, with a fixed capacity of places. Every Swim Session is one of three kinds: a Lane Swim Session, a Public Swim or a Disabled Swim.
_Avoid_: Slot, class, booking

**Lane Swim Session**:
A Swim Session for swimming lengths in one Lane at one Speed.
_Avoid_: Lap swim, lane booking

**Public Swim**:
A Swim Session open to anyone for general swimming, not lengths. It takes over a few Lanes, or sometimes the whole Training Pool.
_Avoid_: Family swim, general swim, fun session

**Disabled Swim**:
A Swim Session set aside for disabled swimmers.
_Avoid_: Accessible swim

**Lane**:
One lane of a Pool, given over to a Swim Session.

**Speed**:
The pace a Lane Swim Session's Lane is set aside for: Fast, Medium or Slow. Other Swim Sessions have no Speed.
_Avoid_: Level, lane type

**Cancelled Session**:
A Swim Session that was published and then withdrawn before its start time, e.g. for a gala, a bank holiday or maintenance. A Swim Session that disappears after it has started is just over, not cancelled.
_Avoid_: Closure

### Busyness

**Busyness**:
The fraction of a Swim Session's places that are booked. It measures bookings, not how many people are in the water.
_Avoid_: Occupancy, attendance, headcount

**Pool Busyness**:
The Busyness of a Pool at a moment in time: places booked across every Lane Swim Session in that Pool running then, divided by their combined capacity. Public Swims and Disabled Swims don't count.

**Speed Busyness**:
The Busyness of a Pool's Lane Swim Sessions at one Speed at a moment in time: places booked across those running then, divided by their combined capacity. Cancelled Sessions don't count.

**Final Busyness**:
A Swim Session's Busyness at its start time.

**Typical Busyness**:
The median Final Busyness over the last 8 weeks of Lane Swim Sessions in the same Pool, at the same Speed, on the same weekday, at the same start time. Cancelled Sessions are left out.
_Avoid_: Average, usual
