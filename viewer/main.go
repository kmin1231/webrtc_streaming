package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

var serverAddr = "ws://localhost:8080/signal"

func main() {
	// WebRTC PeerConnection 설정
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
	defer peerConnection.Close()

	// 데이터 채널 처리
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("New DataChannel %s %d\n", dc.Label(), dc.ID())
	})

	// 연결 상태 모니터링 - 더 자세한 로깅 추가
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Connection state changed: %s\n", state.String())
		switch state {
		case webrtc.PeerConnectionStateConnected:
			log.Println("WebRTC 연결 성공!")
		case webrtc.PeerConnectionStateDisconnected:
			log.Println("WebRTC 연결 끊김")
		case webrtc.PeerConnectionStateFailed:
			log.Println("WebRTC 연결 실패")
		}
	})

	// 트랙 수신 처리 개선
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		codec := track.Codec()
		log.Printf("새로운 트랙 수신: %s (PayloadType: %d)\n", codec.MimeType, codec.PayloadType)

		// 트랙 타입에 따른 처리
		switch codec.MimeType {
		case webrtc.MimeTypeVP8:
			log.Println("비디오 트랙(VP8) 수신 시작")
		default:
			log.Printf("알 수 없는 코덱 타입: %s\n", codec.MimeType)
		}

		buf := make([]byte, 1500)
		for {
			n, _, err := track.Read(buf)
			if err != nil {
				log.Printf("트랙 읽기 오류: %v", err)
				return
			}

			log.Printf("데이터 수신 (크기: %d bytes): %s", n, string(buf[:n]))

			// RTCP 송신자 리포트 전송
			if err := peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.ReceiverReport{},
			}); err != nil {
				log.Printf("RTCP 패킷 전송 오류: %v", err)
			}
		}
	})

	// WebSocket 연결
	log.Printf("시그널링 서버 연결 중: %s", serverAddr)
	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, http.Header{})
	if err != nil {
		log.Fatalf("시그널링 서버 연결 실패: %v", err)
	}
	defer conn.Close()
	log.Println("시그널링 서버 연결 성공")

	// 역할 식별 메시지 전송
	err = conn.WriteMessage(websocket.TextMessage, []byte("viewer"))
	if err != nil {
		log.Fatalf("역할 메시지 전송 실패: %v", err)
	}
	log.Println("Viewer 역할 메시지 전송 완료")

	// ICE Candidate 처리
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("새로운 로컬 ICE candidate: %v", candidate.String())
			candidateJSON, err := json.Marshal(candidate.ToJSON())
			if err != nil {
				log.Printf("ICE candidate 마샬링 실패: %v", err)
				return
			}
			err = conn.WriteMessage(websocket.TextMessage, candidateJSON)
			if err != nil {
				log.Printf("ICE candidate 전송 실패: %v", err)
			}
		}
	})

	// ICE 연결 상태 모니터링
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE 연결 상태 변경: %s\n", state.String())
		switch state {
		case webrtc.ICEConnectionStateChecking:
			log.Println("ICE Checking...")
		case webrtc.ICEConnectionStateConnected:
			log.Println("ICE Connected!")
		case webrtc.ICEConnectionStateFailed:
			log.Println("ICE Connection Failed")
		}
	})

	// Signaling 메시지 처리
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("시그널링 서버 오류: %v", err)
				return
			}

			log.Printf("시그널링 서버로부터 메시지 수신")

			// ICE candidate 확인 및 처리
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal(message, &candidate); err == nil && candidate.Candidate != "" {
				log.Printf("ICE candidate 수신: %v", candidate.Candidate)
				err = peerConnection.AddICECandidate(candidate)
				if err != nil {
					log.Printf("ICE candidate 추가 실패: %v", err)
				}
				continue
			}

			// SDP 처리
			var sdp webrtc.SessionDescription
			if err := json.Unmarshal(message, &sdp); err != nil {
				log.Printf("SDP 파싱 실패: %v", err)
				continue
			}

			log.Printf("SDP 타입 수신: %s", sdp.Type.String())

			if sdp.Type == webrtc.SDPTypeOffer {
				log.Println("Offer 수신, Remote Description 설정")
				err = peerConnection.SetRemoteDescription(sdp)
				if err != nil {
					log.Printf("Remote Description 설정 실패: %v", err)
					continue
				}

				// Answer 생성
				log.Println("Answer 생성 중")
				answer, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					log.Printf("Answer 생성 실패: %v", err)
					continue
				}

				log.Println("Local Description 설정 중")
				err = peerConnection.SetLocalDescription(answer)
				if err != nil {
					log.Printf("Local Description 설정 실패: %v", err)
					continue
				}

				answerJSON, err := json.Marshal(answer)
				if err != nil {
					log.Printf("Answer 마샬링 실패: %v", err)
					continue
				}

				log.Println("Answer를 시그널링 서버로 전송")
				err = conn.WriteMessage(websocket.TextMessage, answerJSON)
				if err != nil {
					log.Printf("Answer 전송 실패: %v", err)
					continue
				}
				log.Println("Answer 전송 완료")
			}
		}
	}()

	// 프로그램 종료 방지
	select {}
}
