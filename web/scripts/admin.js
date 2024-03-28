var jsonData;

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws";
    let conn = new WebSocket(url);

    conn.onmessage = function (evt) {
        StartHeartbeat();
        try {
            jsonData = JSON.parse(evt.data);
            $("#system").text(jsonData.System);
            document.title = jsonData.System;
            $("#version").text(jsonData.Version);
            leakDetection(jsonData.SystemAlarms.h2Alarm);
            conductivityDetection(jsonData.SystemAlarms.conductivityAlarm);
            jsonData.Relays.Relays.forEach(UpdateRelay);
            jsonData.DigitalIn.Inputs.forEach(UpdateInput);
            jsonData.DigitalOut.Outputs.forEach(UpdateOutput);
            jsonData.Analog.Inputs.forEach(UpdateAnalog);
            jsonData.Buttons.forEach(UpdateButton);
            $("#temperature").html(jsonData.Analog.GasTemperature + "&deg;C");
            if (jsonData.kgH2 > 1.0) {
                $("#kgH2").text(jsonData.kgH2.toFixed(3) + "kg");
            } else {
                $("#kgH2").text((jsonData.kgH2 * 1000).toFiged(0) + "g")
            }
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
            let dialCount = jsonData.Electrolysers.length;
            let gridTemplate = "";
            fcStackPower = $("#fcStackPower");
            StackContainer = $("#StackContainer");
            Systems = $("#systems");
            if (jsonData.PanFuelCellStatus != null) {
                dialCount++;
                for (i = 0; i < dialCount; i++) {
                    gridTemplate += " " + (100 / dialCount) + "%";
                }
                Systems.css("grid-template-columns: 20% 20% 20% 20% 20%")
                $("#BMSPower").text(jsonData.PanFuelCellStatus.BMSPower);
                $("#BMSHigh").text(jsonData.PanFuelCellStatus.BMSHigh);
                $("#BMSLow").text(jsonData.PanFuelCellStatus.BMSLow);
                $("#BMSCurrentPower").text(jsonData.PanFuelCellStatus.BMSCurrentPower);
                $("#BMSTargetPower").text(jsonData.PanFuelCellStatus.BMSTargetPower);
                $("#BMSTargetHigh").text(jsonData.PanFuelCellStatus.BMSTargetHigh);
                $("#BMSTargetLow").text(jsonData.PanFuelCellStatus.BMSTargetLow);
                $("#FCStatus").text(jsonData.PanFuelCellStatus.RunStatus);
                $("#FCDCOutputStatus").text(jsonData.PanFuelCellStatus.DCOutputStatus);
                let gauge = {
                    caption: {
                        value: 'Stack Power ' + jsonData.PanFuelCellStatus.StackPower.toFixed(1) + ' kW'
                    },
                    width: "100%" //,
//                    height: "100%"
                };
                fcStackPower.jqxGauge(gauge);

                if ((jsonData.PanFuelCellStatus.RunStatus === 'Start') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'Hydrogen intake') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'AirPurge') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'Hydrogen leak check') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'manual')) {
                    fcStackPower.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }, width: "100%"});
                } else {
                    fcStackPower.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }, width: "100%"});
                }

                fcStackPower.val(jsonData.PanFuelCellStatus.StackPower);
                showFuelCellAlarms(jsonData.PanFuelCellStatus, $("#fcAlarms"));
            } else {
                fcStackPower.hide();
                for (i = 0; i < dialCount; i++) {
                    gridTemplate += " " + (100 / dialCount) + "%";
                }
            }
            Systems.css({"grid-template-columns": gridTemplate});
            jsonData.Electrolysers.forEach((currentElement, index) => updateElectrolyser(currentElement, index));

            if (jsonData.Power !== null) {
                jsonData.Power.forEach(UpdatePower);
            }

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

function openElectrolyser(name, id) {

    if ($("#"+id).attr("on") === "1") {
        window.open("Electrolyser.html?name=" + name);
    } else {
        window.open("ElectrolyserData.html?name=" + name);
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
//    size = smallest(stackPower.width(), stackPower.height());
    stackPower.jqxGauge({
        // height: size,
        // width: size,
        // radius: (size / 2) - 25,
        ticksMinor: {interval: 500, size: '5%'},
        ticksMajor: {interval: 1000,size: '9%'},
        labels: {interval:5000},
        min: 0,
        max: 15000,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Stack Power 0.0 kW', position: 'bottom', offset: [0, 10], visible: true},
    });

    $(window).resize(function () {
        if (this.resizeTO) clearTimeout(this.resizeTO);
        this.resizeTO = setTimeout(function () {
            $(this).trigger('windowResize');
        }, 500);
    });

    RegisterWebSocket();
}

function AddSource (source) {
    strRow = '<tr class="PowerData" id="sourceRow%d"  ondblclick="openPowerChart"><td class="PowerName"><span id="source%d"></span></td><td class="PowerData"><span id="iBatt%d"></span></td><td class="PowerData"><span id="vBatt%d"></span></td><td class="PowerData"><span id="socBatt%d"></span></td><td class="PowerData"><span id="solar%d"></span></td><td class="PowerData"><span id="hz%d"></span></td></tr>';
    let rows = 0;
    let src = "#source";
    for (; $(src).length; rows++){

    }
    strRow = strRow.replaceAll("%d", rows);
    $("#PowerInputs tr:last").after(strRow);
    $("#source"+rows).text(source);
    return rows
}

function openPowerChart(event) {
    let source = event.target.id.replace("Row","");
    let url = 'PowerData.html?source=' + encodeURIComponent($(source).text());
    window.open( url );
}

function findPowerRow (source) {
    let row = 0;
    for (;;) {
        let src = $("#source" + row);
        if (src.length) {
            if (src.text() === source) {
                return row;
            }
            row++;
        } else {
            return AddSource(source);
        }
    }
}

function UpdatePower(Power) {
    idx = findPowerRow(Power.source)
    $("#hz"+idx).text((Power.hz).toFixed(1) + "hz");
    $("#iBatt"+idx).text((Power.amps).toFixed(2) + "A");
    $("#vBatt"+idx).text((Power.volts).toFixed(2) + "V");
    $("#socBatt"+idx).text((Power.soc).toFixed(2) + "%");
    $("#solar"+idx).text((Power.solar / 1000).toFixed(3) + "kW");
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

function UpdateButton(Button, idx) {
    td_button = $("#button" + idx);
    if (Button.Name.startsWith("Button-")) {
        td_button.hide();
    } else {
        if (Button.Pressed) {
            td_button.removeClass("ButtonOff");
            td_button.removeClass("ButtonChanging");
            td_button.addClass("ButtonOn");
        } else {
            td_button.removeClass("ButtonOn");
            td_button.removeClass("ButtonChanging");
            td_button.addClass("ButtonOff");
        }
        $("#buttonText"+idx).text(Button.Name);
        td_button.show();
    }
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

function setOnOffButton(id, on) {
    btn = $("#"+id);
    if (on) {
        if (!btn.hasClass("swOn")) {
            btn.removeClass("swOff");
            btn.addClass("swOn");
        }
    } else {
        if (!btn.hasClass("swOff")) {
            btn.removeClass("swOn");
            btn.addClass("swOff");
        }
    }
}

// function  PowerClick(id, controlID) {
//     let Control = $("#"+controlID);
//     let putString = "/setRelay/" + id + "/" + ((Control.hasClass("swOff")) ? "on" : "off");
//     $.ajax({
//         url: putString,
//         type: 'put',
//         headers: {
//             "Content-Type": "application/json"
//         },
//         dataType: 'json',
//         success: function(response) {
//             console.log("Relay command sent OK");
//         },
//         error: function (xhr, ajaxOptions, thrownError) {
//             if (xhr.status == 400) {
//                 alert(xhr.responseJSON.errors[0].Err);
//             } else {
//                 alert(xhr.status + " : " + thrownError);
//             }
//         }
//     });
// }
//
// function RunClick(id, controlID, elName) {
//     let Control = $("#"+controlID);
//     let putString = "/setElectrolyser/" + ((Control.hasClass("swOff")) ? "Start" : "Stop") + "/" + elName;
//     $.ajax({
//         url: putString,
//         type: 'put',
//         headers: {
//             "Content-Type": "application/json"
//         },
//         dataType: 'json',
//         success: function(response) {
//             console.log("Electrolyser command sent OK");
//         },
//         error: function (xhr, ajaxOptions, thrownError) {
//             if (xhr.status == 400) {
//                 alert(xhr.responseJSON.errors[0].Err);
//             } else {
//                 alert(xhr.status + " : " + thrownError);
//             }
//         }
//     });
// }
