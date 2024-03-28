var jsonData;

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
            document.title = jsonData.System;
            $("#version").text(jsonData.Version);

            leakDetection(jsonData.SystemAlarms.h2Alarm);
            conductivityDetection(jsonData.SystemAlarms.conductivityAlarm);
            jsonData.Buttons.forEach(UpdateButton);

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
            updatePressure(jsonData.Analog.Inputs[jsonData.SystemSettings.gasInput].Value, jsonData.SystemSettings.gasUnits, jsonData.SystemSettings.gasDisplayUnits, jsonData.SystemSettings.gasCapacity);
            updateConductivity(jsonData.Analog.Inputs[7], jsonData.SystemSettings.maxConductivityGreen, jsonData.SystemSettings.maxConductivityYellow);
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

function updatePressure(pressure, units, displayUnits, capacity) {
    let Gas = $("#gas");
    if (Gas.children().length < 1) {
        let gasTitle = $("#gasTitle");
        switch (displayUnits)  {
            case "bar" :
                Gas.jqxLinearGauge({
                    max: 35,
                    min: 0,
                    height: "80%",
                    width: "100%",
                    colorScheme: 'scheme02',
                    ticksPosition: 'far',
                    ticksOffset: [40, 15],
                    ticksMajor: { size: '10%', interval: 10 },
                    ticksMinor: { size: '5%', interval: 5, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                    labels: { interval: 5, position: "far"  },
                    ranges: [
                        { startValue: 0, endValue: 10, style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: 10, endValue: 25, style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: 25, endValue: 35, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("H2 Pressure (Bar)");
                break;
            case "psi" :
                Gas.jqxLinearGauge({
                    max: 500,
                    min: 0,
                    height: "80%",
                    colorScheme: 'scheme02',
                    ticksPosition: 'far',
                    ticksOffset: [30, 15],
                    ticksMajor: { size: '10%', interval: 100 },
                    ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                    labels: { interval: 100, position: 'far' },
                    ranges: [
                        { startValue: 0, endValue: 100, style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: 100, endValue: 300, style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: 300, endValue: 500, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("H2 Pressure (PSI)");
                break;
            case "litres" :
                Gas.jqxLinearGauge({
                    max: capacity,
                    min: 0,
                    height: "80%",
                    colorScheme: 'scheme02',
                    ticksPosition: 'far',
                    ticksOffset: [20, 15],
                    ticksMajor: { size: '10%', interval: 100 },
                    ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                    labels: { interval: Math.round(capacity / 5), position: 'far' },
                    ranges: [
                        { startValue: 0, endValue: Math.round(capacity /5), style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: Math.round(capacity /5), endValue: Math.round(capacity /5) * 3, style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: Math.round(capacity /5) * 3, endValue: capacity, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("H2 Volume (litres)");
                break;
            case "cuft" :
                let cuft = capacity * 0.0353; // Conversion from litres to cubic feet
                Gas.jqxLinearGauge({
                    max: cuft,
                    min: 0,
                    height: "80%",
                    colorScheme: 'scheme02',
                    ticksPosition: 'far',
                    ticksOffset: [20, 15],
                    ticksMajor: { size: '10%', interval: 100 },
                    ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                    labels: { interval: Math.round(cuft / 5),
                        position: 'far',
                        formatValue: function(value){return Math.round(value);},
                    },

                    ranges: [
                        { startValue: 0, endValue: Math.round(cuft / 5), style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: Math.round(cuft / 5), endValue: Math.round(cuft / 5) * 3, style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: Math.round(cuft / 5) * 3, endValue: cuft, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("H2 Volume (cubic feet)");
                break;
            case "kWhr" :
                let kWhr = Math.round((capacity / 990) + 0.5); // Conversion from litres to kWhr
                Gas.jqxLinearGauge({
                    max: kWhr,
                    min: 0,
                    height: "80%",
                    colorScheme: 'scheme02',
                    ticksPosition: 'far',
                    ticksOffset: [40, 15],
                    ticksMajor: { size: '10%', interval: 100 },
                    ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                    labels: { interval: Math.round(kWhr / 5),
                              position: 'far',
                              formatValue: function(value){return Math.round(value);},
                    },
                    ranges: [
                        { startValue: 0, endValue: Math.round(kWhr / 5), style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: Math.round(kWhr / 5), endValue: Math.round(kWhr / 5) * 3, style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: Math.round(kWhr / 5) * 3, endValue: kWhr, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("Energy Stored (kWhr)");
                break;
        }
    }
    if (units === "bar") {
        switch (displayUnits) { // capacity is in litres
            case "litres" : pressure = (pressure / 35) * capacity; // Bar -> litres
                break;
            case "cuft" : pressure = (pressure / 35) * capacity * 0.03535; // Bar -> cubic feet
                break;
            case "kWhr" : pressure = ((pressure / 35) * capacity) / 990; // Bar -> kwHr
                break;
            case "psi" : pressure = pressure * 14.5
        }
    } else if (units === "psi") {
        switch (displayUnits) { // capacity is in litres
            case "litres" : pressure = (pressure / 507.6) / capacity; // psi -> litres
                break;
            case "cuft" : pressure = (pressure / 507.6) * capacity * 0.03535; // psi -> cubic feet
                break;
            case "kWhr" : pressure = ((pressure / 507.6) * capacity) / 990; // psi -> kwHr
                break;
            case "bar" : pressure = pressure * 0.0689 // psi -> bar
        }
    }
    Gas.val(pressure.toFixed(0))
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

