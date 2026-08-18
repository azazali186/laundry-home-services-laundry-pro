package server

import "time"

// timeNow is indirected so tests can stub time.
var timeNow = time.Now
