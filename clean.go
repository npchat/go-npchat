package main

import (
	"fmt"
	"time"
)

func CleanStore(hc HousekeepingChannels, period time.Duration) {
	for {
		hc.GetKeys <- true
		keys := <-hc.Keys
		for _, k := range keys {
			hc.GetMsgsForKey <- k
			// append messages to new set where Time < now
			msgs := []StorableMessage{}
			for m := range hc.MsgsForKey {
				if m.Body == nil {
					// sends an empty message to signal done
					break
				}
				if m.Time.Before(time.Now()) {
					msgs = append(msgs, m)
				} else {
					fmt.Println(k, "-> kicked out")
				}
			}
			hc.StoreKeyValue <- StoreKeyValue{Id: k, Msgs: msgs}
		}
		time.Sleep(period)
	}
}
