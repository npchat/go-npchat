package main

import (
	"time"
)

func CleanStore(hc HousekeepingChannels, period time.Duration) {
	for {
		hc.GetKeys <- true
		keys := <-hc.Keys
		for _, k := range keys {
			hc.GetMsgsForKey <- k
			msgs := []StorableMessage{}
			// append messages to new set where Time < now
			for m := range hc.MsgsForKey {
				if m.Body == nil {
					break
				}
				if m.Time.Before(time.Now()) {
					msgs = append(msgs, m)
				}
			}
			hc.StoreKeyValue <- StoreKeyValue{Id: k, Msgs: msgs}
		}
		time.Sleep(period)
	}
}
