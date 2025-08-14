let jsonData;
let wsConn;

// Add the fuel cell gauge
function AddFuelCell() {
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
        caption: {value: 'Stack Power ' + jsonData.PanFuelCellStatus.StackPower.toFixed(1) + ' kW', position: 'bottom', offset: [0, 10], visible: true},
    });
    return FuelCell;
}

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
            $("#system").text(jsonData.System);
            document.title = jsonData.System;
            $("#version").text(jsonData.Version);

            leakDetection(jsonData.SystemAlarms.h2Alarm);
            conductivityDetection(jsonData.SystemAlarms.conductivityAlarm);
            jsonData.Buttons.forEach(UpdateButton);
            jsonData.DigitalIn.Inputs.forEach(UpdateInput);

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
                    FuelCell = AddFuelCell();
                }
                FuelCell.show();
                if ((jsonData.PanFuelCellStatus.RunStatus === 'Start') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'Hydrogen intake') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'AirPurge') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'Hydrogen leak check') ||
                    (jsonData.PanFuelCellStatus.RunStatus === 'manual')) {
                    FuelCell.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }, width: "100%"});
                } else {
                    FuelCell.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }, width: "100%"});
                }
                FuelCell.val(jsonData.PanFuelCellStatus.StackPower);
                FuelCell.jqxGauge({width:"100%"})
                showFuelCellAlarms(jsonData.PanFuelCellStatus, $("#fcAlarms"));
            }
            jsonData.Electrolysers.forEach((currentElement, index) => updateElectrolyser(currentElement, index));
//            updatePressure(jsonData.Analog.Inputs[jsonData.SystemSettings.gasInput].Value, jsonData.SystemSettings.gasUnits, jsonData.SystemSettings.gasDisplayUnits, jsonData.SystemSettings.gasCapacity);
//            updatePressure(jsonData.Analog.Inputs[jsonData.SystemSettings.gasInput].Value, jsonData.SystemSettings.gasDisplayUnits, jsonData.SystemSettings.gasCapacity);
            updatePressure(jsonData.h2);
            updateConductivity(jsonData.Analog.Inputs[7], jsonData.SystemSettings.maxConductivityGreen, jsonData.SystemSettings.maxConductivityYellow);
            RenderButtons(jsonData.Buttons);
        } catch (e) {
//            alert(e);
            $("#ErrorText").text(e);
            console.log(e);
        }
    }
}

function UpdateButton(Button, idx) {
    td_button = $("#button" + idx);
    if (Button.Name.startsWith("Button-") || !Button.ShowOnCustomer) {
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

function UpdateInput(Input, idx) {
    td_input = $("#di" + idx);
    if (Input.Name.startsWith("Input-") || !Input.ShowOnCustomer) {
        td_input.hide();
    } else {
        if (Input.Value) {
            td_input.removeClass("DILow");
            td_input.addClass("DIHigh");
        } else {
            td_input.removeClass("DIHigh");
            td_input.addClass("DILow");
        }
        $("#InputText"+idx).text(Input.Name);
        td_input.show();

    }
}


function openElectrolyser(name, id) {
    // Do nothing in the user interface, only used in the admin interface
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

//function updatePressure(pressure, units, displayUnits, capacity) {
function updatePressure(h2) {
    let Gas = $("#gas");
    if (Gas.children().length < 1) {
        let gasTitle = $("#gasTitle");
        let gaugeSettings = {
            max : Math.round(h2.maxPressure),
            min: 0,
            height: "80%",
            colorScheme: 'scheme02',
            ticksPosition: 'far',
            ticksOffset: [40, 15],
            ticksMajor: { size: '10%', interval: h2.maxPressure / 5 },
            ticksMinor: { size: '5%', interval: h2.maxPressure / 10, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
            labels: { interval: Math.round(h2.maxPressure / 5), position: "far"  },
            ranges: [
                { startValue: 0, endValue: h2.maxPressure * 0.25, style: { fill: '#FF4800', stroke: '#FF4800'} },
                { startValue: h2.maxPressure * 0.25, endValue: h2.maxPressure * 0.7, style: { fill: '#FFA200', stroke: '#FFA200'}},
                { startValue: h2.maxPressure * 0.7, endValue: h2.maxPressure, style: { fill: '#00B000', stroke: '#00B000'}}],
            pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
            value: 0,
            animationDuration: 0,
        }
        Gas.jqxLinearGauge(gaugeSettings);
        gasTitle.text("H2 Pressure (" + h2.pressureUnits + ")");
    }
    Gas.val(h2.pressure);

}

function updateConductivity(conductivity, greenMax, yellowMax) {
    let cond = $("#conductivity");
    if (cond.children().length < 1) {
        cond.width("100%");
        cond.jqxLinearGauge({
            max: yellowMax + greenMax,
            min: 0,
            width: "150px",
            height: "30px",
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
    cond.val(conductivity.Value.toFixed(1))
}

