var jsonData;
var lockSliders = false;

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws";
    let conn = new WebSocket(url);

    conn.onmessage = function (evt) {
        if (wstimeout !== 0) {
            clearTimeout(wstimeout);
        }

        StartHeartbeat();

        try {
            jsonData = JSON.parse(evt.data);
            $("#system").text(jsonData.System);
            $("#version").text(jsonData.Version);

            leakDetection(jsonData.SystemAlarms.h2Alarm);
            conductivityDetection(jsonData.SystemAlarms.conductivityAlarm);

            let numDials = jsonData.Electrolysers.length + (jsonData.PanFuelCellStatus != null ? 1 : 0);

            let width = Math.floor(100 / ((numDials * 2) + 1))
            let columnSplit = width + "%";
            for (i = 0; i < numDials; i++) {
                columnSplit += " " + (width * 2) + "%";
            }
            $("#systems").css("grid-template-columns", columnSplit);
            if (jsonData.PanFuelCellStatus !== null) {
                let FuelCell = $("#fcStackPower");
                if (FuelCell.length === 0) {
                    sCode = `<div id="FuelCell" class="centered" style="{visibility:hidden}"><div id="fcStackPower"></div></div>`;
                    jQuery('#systems').append(sCode);

                    FuelCell = $("#fcStackPower");
                    FuelCell.jqxGauge({
                        ticksMinor: {interval: 500, size: '5%'},
                        ticksMajor: {interval: 1000,size: '9%'},
                        labels: {interval:5000, position: "far"},
                        min: 0,
                        max: 15000,
                        value: 0,
                        animationDuration: 500,
                        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
                        caption: {value: 'Stack Power', position: 'bottom', offset: [0, 10], visible: true},
                    });
                }
                $("FuelCell").show();
                FuelCell.val(jsonData.PanFuelCellStatus.StackPower);
                FuelCell.jqxGauge({width:"100%"})
            }
            jsonData.Electrolysers.forEach((currentElement, index) => {
                if ($("#el" + index).length === 0) {
                    addElectrolyser(index, currentElement.name, currentElement.powerRelay);
                    newEl = $("#el" + index);
                    newEl.jqxGauge({
                        ticksMinor: {interval: 25, size: '5%'},
                        ticksMajor: {interval: 100,size: '9%'},
                        labels: {interval:100, position: "far" },
                        min: 0,
                        max: 525,
                        value: 0,
                        animationDuration: 500,
                        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
                        caption: {value: currentElement.name + ' NL/hr', position: 'bottom', offset: [0, 10], visible: true},
                    });
                    rate = $('#elRate' + index);
                    rate.jqxSlider(
                        {
                            theme: "energyblue",
                            showTickLabels: true,
                            tooltip: true,
                            mode: "fixed",
                            min: 60,
                            max: 100,
                            height: 50,
                            width: "100%",
                            ticksFrequency: 10,
                            value: 0,
                            step: 1,
                            orientation: "horizontal",
                            showButtons: false,
                            ticksPosition: 'bottom',
                        }
                    );
                    rate.on('slideEnd', function (event) {
                        setRate(event.args.value, currentElement.name);
                        lockSliders = false;
                    });
                    rate.on('slideStart', function () {
                        lockSliders = true;
                    })
                }
                el = $("#el" + index);
                if (currentElement.on && currentElement.state != 0) {
                    el.attr('on', "1");
                    el.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }, width: "100%"});
                } else {
                    el.attr('on', "0");
                    el.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }, width: "100%"});
                }
                setOnOffButton("ELPower"+index, currentElement.on);
                el.val(currentElement.h2Flow);
                if ((currentElement.state === 3) || (currentElement.state === 4)) {
                    setOnOffButton("ELRun" + index, true)
                } else {
                    setOnOffButton("ELRun" + index, false)
                }
                let stat = $("#ELStatus" + index);
                if (currentElement.on) {
                    switch (currentElement.state) {
                        case 0 : stat.text("Halted");
                            break;
                        case 1 : stat.text("Maintenance mode");
                            break;
                        case 2 : stat.text("Idle");
                            break;
                        case 3 : stat.text("Steady");
                            break;
                        case 4 : stat.text("Standby");
                            break;
                        case 5 : stat.text("Curve");
                            break;
                        case 6 : stat.text("Blow Down");
                            break;
                        default : stat.text("Unknown State");
                    }
                } else {
                    stat.text("Off");
                }
                rate = $('#elRate' + index);
                if (!lockSliders) {
                    rate.val(currentElement.rate);
                }
                if (currentElement.on) {
                    rate.jqxSlider({ disabled:false });
                } else {
                    rate.jqxSlider({ disabled:true });
                }
            });
            updatePressure(jsonData.Analog.Inputs[jsonData.SystemSettings.gasInput].Value, jsonData.SystemSettings.gasUnits);
            updateConductivity(jsonData.Analog.Inputs[7], jsonData.SystemSettings.maxConductivityGreen, jsonData.SystemSettings.maxConductivityYellow);
        } catch (e) {
//            alert(e);
            $("#ErrorText").text(e);
            console.log(e);
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

function setRate(rate, elName) {
    url = "/setElectrolyser/Production/" + elName + "/" + rate;
    $.ajax({
        method : "PUT",
        url: url
    });
}

const ELDIV = `<div class="centered">
    <div class="control" id="el{{id}}">
    </div>
    <div class="control" id="elControls{{id}}">
        <div class="control">
            <label class="parameters" for="elRate{{id}}">Production Rate</label><br \>
            <div style="margin:auto" id="elRate{{id}}"></div>
        </div>
        <div class="control" style="display: grid; grid-template-columns: 40% 20% 40%" >
            <div class="control">
                <img id="ELPower{{id}}" class="swOff" src="images/power-off.png" alt="Enable" onclick="PowerClick({{relay}}, 'ELPower{{id}}')" />
                <label for="ELPower{{id}}">Power</label></td>
            </div>
            <div class="control">
                <span id="ELStatus{{id}}">Off</span>
            </div>
            <div class="control">
                <img id="ELRun{{id}}" class="swOff" src="images/power-off.png" alt="Enable" onclick="RunClick({{id}}, 'ELRun{{id}}', '{{name}}')" />
                <label for="ELRun{{id}}">Run</label></td>
            </div>
        </div>
    </div>
</div>`

function addElectrolyser(id, name, relay) {

    sCode = ELDIV.replace(/{{id}}/g, id).replace(/{{name}}/g, name).replace(/{{relay}}/g, relay);
    jQuery('#systems').append(sCode);
}

function smallest(x, y) {
    if ((x > y) && (y > 0)) {
        return y;
    } else {
        return x;
    }
}

function setupPage() {
    RegisterWebSocket();
}

function updatePressure(pressure, units) {

    if ($("#gas").children().length < 1) {
//        $("#gas").height("100%");
        if (units == "Bar") {
            $("#gas").jqxLinearGauge({
                max: 35,
                min: 0,
                height: "80%",
                pointer: { size: '5%' },
                colorScheme: 'scheme02',
                ticksMajor: { size: '10%', interval: 10 },
                ticksMinor: { size: '5%', interval: 5, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                labels: { interval: 5, position: "far"  },
                ranges: [
                    { startValue: 0, endValue: 10, style: { fill: '#FF4800', stroke: '#FF4800'} },
                    { startValue: 10, endValue: 25, style: { fill: '#FFA200', stroke: '#FFA200'}},
                    { startValue: 25, endValue: 35, style: { fill: '#00B000', stroke: '#00B000'}}],
                pointer: { pointerType: 'default', size: '15%', visible: true, offset: 0 },
                value: 0,
                animationDuration: 0,
            });
        } else {
            $("#gas").jqxLinearGauge({
                max: 500,
                min: 0,
                height: "80%",
                pointer: { size: '5%' },
                colorScheme: 'scheme02',
                ticksMajor: { size: '10%', interval: 100 },
                ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                labels: { interval: 100 },
                ranges: [
                    { startValue: 0, endValue: 100, style: { fill: '#FF4800', stroke: '#FF4800'} },
                    { startValue: 100, endValue: 300, style: { fill: '#FFA200', stroke: '#FFA200'}},
                    { startValue: 300, endValue: 500, style: { fill: '#00B000', stroke: '#00B000'}}],
                pointer: { pointerType: 'default', size: '15%', visible: true, offset: 0 },
                value: 0,
                animationDuration: 0,
            });
        }
    }
    $("#gas").val(pressure.toFixed(0))
}

function updateConductivity(conductivity, greenMax, yellowMax) {

    if ($("#conductivity").children().length < 1) {
        $("#conductivity").width("100%");
        $("#conductivity").jqxLinearGauge({
            max: yellowMax + greenMax,
            min: 0,
            width: "150px",
            height: "30px",
            pointer: {
                pointerType: 'arrow',
                size: '50%'
            },
            colorScheme: 'scheme02',
            ticksMajor: { visible: false },
            ticksMinor: { visible: false },
            labels: { visible: false },
            orientation: 'horizontal',
            rangeSize: '20%',
            ranges: [
                { startValue: 0, endValue: greenMax, style: { fill: '#20C020', stroke: '#20C020'} },
                { startValue: greenMax, endValue: yellowMax, style: { fill: '#F8A000', stroke: '#F8A000'}},
                { startValue: yellowMax, endValue: yellowMax + greenMax, style: { fill: '#E00000', stroke: '#E00000'}}],
            pointer: { pointerType: 'arrow', size: '40%', visible: true, offset: "-50%" },
            value: 0,
            animationDuration: 0,
        });
    }
    $("#conductivity").val(conductivity.Value.toFixed(1))
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

function  PowerClick(id, controlID) {
    let Control = $("#"+controlID);
    let putString = "/setRelay/" + id + "/" + ((Control.hasClass("swOff")) ? "on" : "off");
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function(response) {
            console.log("Relay command sent OK");
        },
        error: function (xhr, ajaxOptions, thrownError) {
            if (xhr.status == 400) {
                alert(xhr.responseJSON.errors[0].Err);
            } else {
                alert(xhr.status + " : " + thrownError);
            }
        }
    });
}

function RunClick(id, controlID, elName) {
    let Control = $("#"+controlID);
    let putString = "/setElectrolyser/" + ((Control.hasClass("swOff")) ? "Start" : "Stop") + "/" + elName;
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function(response) {
            console.log("Electrolyser command sent OK");
        },
        error: function (xhr, ajaxOptions, thrownError) {
            if (xhr.status == 400) {
                alert(xhr.responseJSON.errors[0].Err);
            } else {
                alert(xhr.status + " : " + thrownError);
            }
        }
    });
}
