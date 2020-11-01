package memconn

import "time"

// reduce this to increase performances but also increase cpu usage
const memTiming = 50 * time.Microsecond
