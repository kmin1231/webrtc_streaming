package main

import (
	"encoding/json"
	"net/http"

	"github.com/pion/webrtc/v3"
)

func handleOffer(w http.ResponseWriter, r *http.Request) {
	var offer webrtc.SessionDescription

	// 클라이언트에서 전송된 SDP Offer 디코딩
	err := json.NewDecoder(r.Body).Decode(&offer)
	if err != nil {
		http.Error(w, "Invalid SDP offer", http.StatusBadRequest)
		return
	}

	// PeerConnection 생성
	pc, err := createPeerConnection()
	if err != nil {
		http.Error(w, "Failed to create PeerConnection", http.StatusInternalServerError)
		return
	}

	// Remote SDP 설정
	err = pc.SetRemoteDescription(offer)
	if err != nil {
		http.Error(w, "Failed to set remote description", http.StatusInternalServerError)
		return
	}

	// Answer 생성
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		http.Error(w, "Failed to create SDP answer", http.StatusInternalServerError)
		return
	}

	// Local SDP 설정
	err = pc.SetLocalDescription(answer)
	if err != nil {
		http.Error(w, "Failed to set local description", http.StatusInternalServerError)
		return
	}

	// Answer를 클라이언트로 전송
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(answer)
}
