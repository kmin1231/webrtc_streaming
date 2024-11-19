package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

func main() {
	// Signaling 서버 WebSocket 연결
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/signal"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to signaling server: %v", err)
	}
	defer conn.Close()

	// 역할 식별 메시지 전송
	err = conn.WriteMessage(websocket.TextMessage, []byte("broadcaster"))
	if err != nil {
		log.Fatalf("Failed to send role message: %v", err)
	}

	// WebRTC PeerConnection 구성
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}

	// 비디오 트랙 생성
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeVP8,
		ClockRate: 90000,
	}, "video", "broadcaster")
	if err != nil {
		log.Fatalf("Failed to create video track: %v", err)
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Fatalf("Failed to add track: %v", err)
	}

	// 연결 상태 모니터링
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state changed: %s\n", state.String())
	})

	// ICE 연결 상태 모니터링
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", state.String())
	})

	// Offer 생성 및 로컬 설정
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Fatalf("Failed to create offer: %v", err)
	}
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Fatalf("Failed to set local description: %v", err)
	}

	// Offer를 signaling 서버에 전송
	offerJSON, _ := json.Marshal(offer)
	err = conn.WriteMessage(websocket.TextMessage, offerJSON)
	if err != nil {
		log.Fatalf("Failed to send offer: %v", err)
	}
	fmt.Println("Broadcast Offer sent to signaling server")

	// ICE Candidate 처리
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("New local ICE candidate: %v", candidate.String())
			candidateJSON, _ := json.Marshal(candidate.ToJSON())
			err = conn.WriteMessage(websocket.TextMessage, candidateJSON)
			if err != nil {
				log.Printf("Failed to send ICE candidate: %v", err)
			}
		}
	})

	// Answer 수신 처리
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				return
			}

			var sdp webrtc.SessionDescription
			if err := json.Unmarshal(message, &sdp); err == nil {
				if sdp.Type == webrtc.SDPTypeAnswer {
					log.Println("Received answer, setting remote description")
					err = peerConnection.SetRemoteDescription(sdp)
					if err != nil {
						log.Printf("Failed to set remote description: %v", err)
					}
					log.Println("Remote description set")
				}
			}
		}
	}()

	// 더미 데이터 전송
	go func() {
		sequenceNumber := uint16(0)
		timestamp := uint32(0)

		for {
			// 더미 데이터 생성
			dummyData := []byte(fmt.Sprintf("Frame %d from broadcaster", sequenceNumber))

			// RTP 패킷 생성
			packet := &rtp.Packet{
				Header: rtp.Header{
					Version:        2,
					PayloadType:    96,
					SequenceNumber: sequenceNumber,
					Timestamp:      timestamp,
					SSRC:           12345, // 임의의 SSRC 값
				},
				Payload: dummyData,
			}

			err := videoTrack.WriteRTP(packet)
			if err != nil {
				log.Printf("Failed to write RTP packet: %v", err)
			} else {
				log.Printf("Sent frame %d", sequenceNumber)
			}

			sequenceNumber++
			timestamp += 90000 / 1 // 30 FPS 가정
			time.Sleep(time.Second / 1)
		}
	}()

	// 종료 방지
	select {}
}
