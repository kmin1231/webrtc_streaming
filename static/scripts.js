const pc = new RTCPeerConnection();

navigator.mediaDevices.getUserMedia({ video: true, audio: true })
    .then(stream => {
        stream.getTracks().forEach(track => pc.addTrack(track, stream));
    });

pc.ontrack = (event) => {
    const video = document.createElement('video');
    video.srcObject = event.streams[0];
    video.autoplay = true;
    document.body.appendChild(video);
};

// 시그널링 서버와 연결
const ws = new WebSocket("ws://localhost:8080/signal");

ws.onmessage = async (message) => {
    const data = JSON.parse(message.data);
    if (data.type === "offer") {
        await pc.setRemoteDescription(data);
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        ws.send(JSON.stringify(answer));
    } else if (data.type === "answer") {
        await pc.setRemoteDescription(data);
    }
};
