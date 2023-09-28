var jsonData;

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws";
    let conn = new WebSocket(url);

    conn.onmessage = function (evt) {
        StartHeartbeat();
        try {
            jsonData = JSON.parse(evt.data);
            $("#system").text(jsonData.System);
            $("#version").text(jsonData.Version);
            leakDetection(jsonData.SystemAlarms.h2Alarm);
            conductivityDetection(jsonData.SystemAlarms.conductivityAlarm);
            jsonData.Relays.Relays.forEach(UpdateRelay);
            jsonData.DigitalIn.Inputs.forEach(UpdateInput);
            jsonData.DigitalOut.Outputs.forEach(UpdateOutput);
            jsonData.Analog.Inputs.forEach(UpdateAnalog);
            if (jsonData.ACMeasurements.length > 0) {
                $("#ACMeasurementsDiv").show();
                jsonData.ACMeasurements.forEach(updateAC);
                for (i = jsonData.ACMeasurements.length + 1; i < 5; i++) {
                    $("#AC" + i).hide();
                    $("#ACErr" + i).hide();
                }
            } else {
                $("#ACMeasurementsDiv").hide();
            }
            if (jsonData.DCMeasurements.length > 0) {
                $("#DCMeasurementsDiv").show();
                jsonData.DCMeasurements.forEach(updateDC);
                for (i = jsonData.DCMeasurements.length + 1; i < 5; i++) {
                    $("#DC" + i).hide();
                    $("#DCErr" + i).hide();
                }
            } else {
                $("#DCMeasurementsDiv").hide();
            }
            if (jsonData.PanFuelCellStatus != null) {
                $("#BMSPower").text(jsonData.PanFuelCellStatus.BMSPower);
                $("#BMSHigh").text(jsonData.PanFuelCellStatus.BMSHigh);
                $("#BMSLow").text(jsonData.PanFuelCellStatus.BMSLow);
                $("#BMSCurrentPower").text(jsonData.PanFuelCellStatus.BMSCurrentPower);
                $("#BMSTargetPower").text(jsonData.PanFuelCellStatus.BMSTargetPower);
                $("#BMSTargetHigh").text(jsonData.PanFuelCellStatus.BMSTargetHigh);
                $("#BMSTargetLow").text(jsonData.PanFuelCellStatus.BMSTargetLow);
                $("#FCStatus").text(jsonData.PanFuelCellStatus.RunStatus);
                $("#FCDCOutputStatus").text(jsonData.PanFuelCellStatus.DCOutputStatus);
                $("#fcStackPower").val(jsonData.PanFuelCellStatus.StackPower);
            } else {
                $("#fcStackPower").hide();
            }

            jsonData.Electrolysers.forEach((currentElement, index) => {
                // noinspection JSUnresolvedFunction,JSJQueryEfficiency
                el = $("#el" + index);
                if (el.length === 0) {
                    addElectrolyser(index, currentElement.name);
                    newEl = $("#el" + index);
                    newEl.jqxGauge({
                        height: size,
                        width: size,
                        radius: (size / 2) - 25,
                        ticksMinor: {interval: 25, size: '5%'},
                        ticksMajor: {interval: 100,size: '9%'},
                        labels: {interval:100},
                        min: 0,
                        max: 525,
                        value: 0,
                        animationDuration: 500,
                        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
                        caption: {value: currentElement.name + ' NL/hr', position: 'bottom', offset: [0, 10], visible: true},
                    });
                }
                if (currentElement.on) {
                    el.attr('on', "1");
                    el.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }});
                } else {
                    el.attr('on', "0");
                    el.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }});
                }
                el.val(currentElement.h2Flow);
            });
        } catch (e) {
            alert(e);
        }
    }
}

function leakDetection(alarm) {
    if (alarm) {
        $("#leakAlarmDiv").show();
    } else {
        $("#leakAlarmDiv").hide();
    }
}

function conductivityDetection(alarm) {
    if (alarm) {
        $("#conductivityAlarmDiv").show();
    } else {
        $("#conductivityAlarmDiv").hide();
    }
}

function addElectrolyser(id, name) {
    sCode = '<div class="centered"><div id="el' + id + '" ondblclick="openElectrolyser(\'' + name + '\', event)"></div></div>\n';
    jQuery('#systems').append(sCode);
}

function openElectrolyser(name, event) {

    if ($("#"+event.currentTarget.id).attr("on") === "1") {
        window.open("Electrolyser.html?name=" + name);
    } else {
        alert("Please turn the electrolyser on first.");
    }
}

function openFuelCell() {
    window.open("/FuelCellSettings.html");
    return false;
}

function smallest(x, y) {
    if ((x > y) && (y > 0)) {
        return y;
    } else {
        return x;
    }
}

function setupPage() {
    stackPower = $('#fcStackPower');
    size = smallest(stackPower.width(), stackPower.height());
    stackPower.jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ticksMinor: {interval: 500, size: '5%'},
        ticksMajor: {interval: 1000,size: '9%'},
        labels: {interval:5000},
        min: 0,
        max: 15000,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Stack Power', position: 'bottom', offset: [0, 10], visible: true},
    });

    $(window).resize(function () {
        if (this.resizeTO) clearTimeout(this.resizeTO);
        this.resizeTO = setTimeout(function () {
            $(this).trigger('windowResize');
        }, 500);
    });

    RegisterWebSocket();
}


function UpdateRelay(Relay, idx) {
    td_relay = $("#relay" + idx);
    if (Relay.On) {
        td_relay.removeClass("RelayOff");
        td_relay.removeClass("RelayChanging");
        td_relay.addClass("RelayOn");
    } else {
        td_relay.removeClass("RelayOn");
        td_relay.removeClass("RelayChanging");
        td_relay.addClass("RelayOff");
    }
    $("#relayText"+idx).text(Relay.Name);
}

function UpdateInput(Input, idx) {
    td_input = $("#di" + idx);
    if (Input.Value) {
        td_input.removeClass("DILow");
        td_input.addClass("DIHigh");
    } else {
        td_input.removeClass("DIHigh");
        td_input.addClass("DILow");
    }
    $("#InputText"+idx).text(Input.Name);
}

function UpdateOutput(Output, idx) {
    td_output = $("#do" + idx);
    if (Output.Value) {
        td_output.removeClass("DOLow");
        td_output.addClass("DOHigh");
    } else {
        td_output.removeClass("DOHigh");
        td_output.addClass("DOLow");
    }
    $("#OutputText"+idx).text(Output.Name);
}

function UpdateAnalog(analog, idx) {
    $("#a"+idx+"name").text(analog.Name);
    $("#a"+idx+"raw").text(analog.Raw);
    $("#a"+idx+"value").text(analog.Value.toFixed(2));
}

function updateAC(ac, idx) {
    idx++;
    if (ac.Error === "") {
        $("#AC" + idx).show();
        $("#ACErr" + idx).hide();
        $("#acname" + idx).text(ac.Name);
        $("#acvolts" + idx).text(ac.ACVolts.toFixed(1));
        $("#acamps" + idx).text(ac.ACAmps.toFixed(2));
        $("#acwatts" + idx).text(ac.ACWatts.toFixed(2));
        $("#achz" + idx).text(ac.ACHertz.toFixed(1));
        $("#acpf" + idx).text(ac.ACPowerFactor.toFixed(1));
    } else {
        $("#AC" + idx).hide();
        $("#ACErr" + idx).show();
        $("#acnameerr" + idx).text(ac.Name)
        $("#acerr" + idx).text(ac.Error);
    }
}

function updateDC(dc, idx) {
    idx++;
    if (dc.Error === "") {
        $("#DC" + idx).show();
        $("#DCErr" + idx).hide();
        $("#dcname" + idx).text(dc.Name);
        $("#dcvolts" + idx).text(dc.DCVolts.toFixed( 2));
        $("#dcamps" + idx).text(dc.DCAmps.toFixed(2));
    } else {
        $("#DC" + idx).hide();
        $("#DCErr" + idx).show();
        $("#dcnameerr" + idx).text(dc.Name)
        $("#dcerr" + idx).text(dc.Error);
    }
}


function  clickRelay(id) {
    rl = $("#relay" + id);
    if (rl.hasClass("RelayOff")) {
        action = "on";
    } else if (rl.hasClass("RelayOn")){
        action = "off";
    }
    rl.removeClass("RelayOn");
    rl.removeClass("RelayOff");
    rl.addClass("RelayChanging");
    putString = "/setRelay/"+id+"/" + action
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function() {
            console.log("Relay command sent OK");
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

function  clickOutput(id) {
    op = $("#do" + id);
    if (op.hasClass("DOLow")) {
        action = "on";
    } else {
        action = "off";
    }
    putString = "/setOutput/"+id+"/" + action
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json'
    })
}
