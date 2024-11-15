const localVideo = document.getElementById('localVideo');
const remoteVideo = document.getElementById('remoteVideo');
let localStream;
let peerConnection;

async function startCall() {
    peerConnection = new RTCPeerConnection({
        iceServers: [{ urls: "stun:stun.l.google.com:19302" }]
    });

    // Local stream 가져오기
    localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
    localVideo.srcObject = localStream;
    localStream.getTracks().forEach(track => peerConnection.addTrack(track, localStream));

    // Offer 생성 및 서버로 전송
    const offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);

    const response = await fetch('/offer', {
        method: 'POST',
        body: JSON.stringify(offer),
        headers: { 'Content-Type': 'application/json' },
    });

    const answer = await response.json();
    await peerConnection.setRemoteDescription(answer);

    // Remote track 처리
    peerConnection.ontrack = (event) => {
        remoteVideo.srcObject = event.streams[0];
    };
}
