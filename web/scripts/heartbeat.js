var wstimeout;

function StartHeartbeat() {
    let hb = $("#heartbeat");
    if (hb.length < 1) {
        $("#system").after('<img class="heartbeat" alt="HeartBeat" id="heartbeat" src="images/heartbeat.png" />')
    }
    hb.css({width: "20px", height: "20px"})
    hb.animate({width: "15px", height: "15px"})

    // Clear the timeout timer if it is active
    if (wstimeout !== 0) {
        clearTimeout(wstimeout);
    }
    wstimeout = setTimeout(RegisterWebSocket, 5000)
}
