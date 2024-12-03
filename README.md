# WebRTC
Simple example of WebRTC - WebSocket 
Singnaling 서버에 웹 소켓통신을 통해 서로를 등록하고 WebRTC 연결을 맺음
이후 WebRTC를 통해서 음성, 영상데이터를 주고받는다.

[terminal 1]
```$ go run signaling/server.go```

[terminal 2]
```$ go run viewer/main.go```

[terminal 3]
```$ go run broadcaster/main.go```
