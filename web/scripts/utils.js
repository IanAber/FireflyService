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

function  clickButton(id) {
    rl = $("#button" + id);
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

