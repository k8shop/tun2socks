package tun2socks

import "time"

func timerReset(t *time.Timer, d time.Duration) {
    if !t.Stop() {
        <-t.C
    }
    t.Reset(d)
}
