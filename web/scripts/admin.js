var jsonData;
let wsConn;
let elMap;

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws";
    wsConn = new WebSocket(url);

    wsConn.onmessage = function (evt) {
        if ((Date.now() - lastMessage) < 800) {
            return;
        }
        lastMessage = Date.now();
        StartHeartbeat();
        try {
            jsonData = JSON.parse(evt.data);
            let newMap = "";
            for (let el of jsonData.Electrolysers) {
                newMap = newMap + el.name;
            }
            if (elMap !== undefined) {
                if (newMap !== elMap) {
                    $(".electrolyser").remove();
                }
            } else {
                elMap = newMap;
            }
            $("#system").text(jsonData.System);
            document.title = jsonData.System;
            $("#version").text(jsonData.Version);
            leakDetection(jsonData.SystemAlarms.h2Alarm);
            conductivityDetection(jsonData.SystemAlarms.conductivityAlarm);
            acquiringElectrolysers(jsonData.acquiring)
            jsonData.Relays.Relays.forEach(UpdateRelay);
            jsonData.DigitalIn.Inputs.forEach(UpdateInput);
            jsonData.DigitalOut.Outputs.forEach(UpdateOutput);
            jsonData.Analog.Inputs.forEach(UpdateAnalog);
            RenderButtons(jsonData.Buttons);
            $("#temperature").html(jsonData.Analog.GasTemperature + "&deg;C");
            $("#h2Vol").text(jsonData.h2.volumeText + " " + jsonData.h2.volumeUnits);
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
            StackContainer = $("#StackContainer");
            Systems = $("#systems");
            if (jsonData.PanFuelCellStatus != null) {
                let fcStackPower = $("#fcStackPower");
                StackContainer.show();
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
                StackContainer.hide();
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

function acquiringElectrolysers(status) {
    if (status) {
        $("#acquiringDiv").show();
    } else {
        $("#acquiringDiv").hide();
    }
}

function openElectrolyser(name, id) {

    if ($("#"+id).attr("on") === "1") {
        window.open("Electrolyser.html?name=" + name);
    } else {
        window.open("Electrolyser/Data.html?name=" + name);
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
    MonitorWebService();
}

function AddSource (source) {

    let powerSourceTable = $("#PowerInputs");
    let sourceNum;

    if (powerSourceTable.length === 0) {
        $("#PowerDiv").html(`
    <h2 class="firefly PowerInputs" > Power Management </h2>
    <table id="PowerInputs" class="PowerInputs">
        <thead class="PowerHeader">
            <tr class="PowerHeader">
                <th class="PowerName" colSpan="5">Battery</th>
                <th class="PowerName" colSpan="2">Mains</th>
            </tr>
            <tr class="PowerHeader">
                <th class="PowerName">Source</th>
                <th class="PowerName">Current</th>
                <th class="PowerName">Voltage</th>
                <th class="PowerName">State of Charge</th>
                <th class="PowerName">Maximum Charge Current</th>
                <th class="PowerName">Solar Power</th>
                <th class="PowerName">Frequency</th>
            </tr>
        </thead>
        <tbody class="PowerData">
            <tr class="PowerData PowerRow" ondblclick="openPowerChart(0)">
                <td class="PowerName"><span id="source0">` + source + `</span></td>
                <td class="PowerData"><span id="iBatt0">0.0</span></td>
                <td class="PowerData"><span id="vBatt0">0.0</span></td>
                <td class="PowerData"><span id="socBatt0">0.0</span></td>
                <td class="PowerData"><span id="maxChargeAmps0">0.0</span></td>
                <td class="PowerData"><span id="solar0">0.0</span></td>
                <td class="PowerData"><span id="hz0">0.0</span></td>
            </tr>
        </tbody>
    </table>`);
        sourceNum = 0;
    } else {
       sourceNum = $(".PowerRow").length;
       let strRow = `<tr class="PowerData PowerRow" ondblclick="openPowerChart(${sourceNum})"><td class="PowerName"><span id="source${sourceNum}">${source}</span></td><td class="PowerData"><span id="iBatt${sourceNum}"></span></td><td class="PowerData"><span id="vBatt${sourceNum}"></span></td><td class="PowerData"><span id="socBatt${sourceNum}"></span></td><td class="PowerData"><span id="maxChargeAmps${sourceNum}"></span></td><td class="PowerData"><span id="solar${sourceNum}"></span></td><td class="PowerData"><span id="hz${sourceNum}"></span></td></tr>`;
       $("tbody.PowerData").append(strRow);
    }
    return sourceNum;
}

function openPowerChart(idx) {
    let source = $("#source"+idx).text();
    let url = 'PowerData.html?source=' + encodeURIComponent(source);
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
    $("#maxChargeAmps"+idx).text((Power.bmsChargeCurrentMax).toFixed(2) + "A");
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

function RenderButtons(buttons) {
    const controls = $("#buttonsDiv");
    let buttonCount = 0;
    for (let button of buttons) {
        if (button.Name !== "")
            buttonCount++
    }
    if (controls.children().length === buttonCount) {
        let buttonId = 0;
        for (let button of buttons) {
            if (button.Name !== "") {
                const btn = $("#buttonDiv" + buttonId);

                if (button.Pressed) {
                    btn.removeClass("ButtonChanging");
                    btn.removeClass("ButtonOff");
                    btn.addClass("ButtonOn");
                } else {
                    btn.removeClass("ButtonChanging");
                    btn.removeClass("ButtonOn");
                    btn.addClass("ButtonOff");
                }
            }
            buttonId++;
        }
    } else {
        controls.children().remove();
        let buttonId = 0;
        let buttonTag;
        for (let button of buttons) {
            if (button.Name !== "") {
                let buttonClass = button.Pressed ? 'ButtonOn' : 'ButtonOff'
                buttonTag = `<div class="button ${buttonClass}" onclick="clickButton(${buttonId})" id="buttonDiv${buttonId}"><span class="button" id="button${buttonId}">${button.Name}</span></div>`;
                controls.append(buttonTag);
                const btn = document.getElementById("buttonDiv" + buttonId);
                btn.addEventListener("contextmenu", (e) => {
                    e.preventDefault()
                });
            }
            buttonId++;
        }
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
