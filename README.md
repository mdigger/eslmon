FreeSWITCH ESL Monitor
----------------------

```golang
ch1 := make(chan esl.Event, 1)
ch2 := make(chan esl.Event, 1)

monitor := esl.
    New("127.0.0.1:8021", "ClueCon").
    Subscribe(ch1). // subscribe to all events
    Subscribe(ch2, 
        "CHANNEL_CREATE", 
        "CHANNEL_HANGUP", 
        "CHANNEL_HANGUP_COMPLETE",
    )

err := monitor.Run(context.TODO())
```