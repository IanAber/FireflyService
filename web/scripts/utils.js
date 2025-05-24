function PopulateTitle() {
    fetch("/title")
        .then(function (response) {
            if (response.status === 200) {
                response.json()
                    .then(function (title) {
                        $("#system").text(title.title);
                    })
                    .catch(function (err) {
                        if (err.name === "TypeError" && err.message !== "cancelled") {
                            alert('System Title Fetch Error :-S' + err.message);
                        }
                    });
            }
        });
}

let lastMessage = new(Date);

function StartHeartbeat() {
    $("header").removeClass("alarm");
    let hb = $("#heartbeat");
    if (hb.length < 1) {
        $("#system").after('<img class="heartbeat" alt="HeartBeat" id="heartbeat" src="/images/heartbeat.png" />')
    }
    hb.stop(true,true);
    hb.animate({width: "15px", height: "15px"}, 400, function () { $(this).removeAttr('style'); })
}

function MonitorWebService() {
    setInterval(function(){
        if ((Date.now() - lastMessage) > 5000) {
            $("header").addClass("alarm");
            wsConn.close();
            RegisterWebSocket();
        }
    }, 5000);
}

function  clickButton(id) {
    let rl = $("#buttonDiv" + id);
    if (rl == null) {
        rl = $("#button" + id);
    }
    if (rl.hasClass("ButtonOff")) {
        action = "on";
    } else if (rl.hasClass("ButtonOn")){
        action = "off";
    }
    rl.removeClass("ButtonOn");
    rl.removeClass("ButtonOff");
    rl.addClass("ButtonChanging");
    putString = "/setButton/"+id+"/" + action
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function() {
            console.log("Button command sent OK");
        },
        error: function (xhr, ajaxOptions, thrownError) {
            if (xhr.status === 400) {
                alert(xhr.responseJSON.errors[0].Err);
            } else {
                alert(xhr.status + " : " + thrownError);
            }
        }
    });
}

