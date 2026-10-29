# Read availability from Everyone Active's OpenActive feed

Busyness comes from Everyone Active's public OpenActive feed (CC BY 4.0, so pages showing it credit Everyone Active), read incrementally every 5 minutes. Every change to a session's places left is stored, rather than snapshots at fixed intervals. The app's own API has richer data (waitlists), but it needs a key extracted from the iOS app, which is a terms-of-service risk, so we don't use it.

## Consequences

The feed's place counts for London Aquatics Centre showed no bookings when first checked, while other centres' counts did. The feed must be watched for 48 hours before anything is built on it. If the counts never move, switch to scraping the Connect booking portal with a member login, read-only. The portal may only say available, waitlist or full, in which case Busyness can't be a fraction and needs redefining.
