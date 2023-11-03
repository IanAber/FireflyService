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
            jsonData.Electrolysers.forEach((currentElement, index) => updateElectrolyser(currentElement, index));
            updatePressure(jsonData.Analog.Inputs[jsonData.SystemSettings.gasInput].Value, jsonData.SystemSettings.gasUnits);
            updateConductivity(jsonData.Analog.Inputs[7], jsonData.SystemSettings.maxConductivityGreen, jsonData.SystemSettings.maxConductivityYellow);
        } catch (e) {
//            alert(e);
            $("#ErrorText").text(e);
            console.log(e);
        }
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

function updatePressure(pressure, units) {
    let Gas = $("#gas");
    if (Gas.children().length < 1) {
//        $("#gas").height("100%");
        if (units === "Bar") {
            Gas.jqxLinearGauge({
                max: 35,
                min: 0,
                height: "80%",
                colorScheme: 'scheme02',
                ticksMajor: { size: '10%', interval: 10 },
                ticksMinor: { size: '5%', interval: 5, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                labels: { interval: 5, position: "far"  },
                ranges: [
                    { startValue: 0, endValue: 10, style: { fill: '#FF4800', stroke: '#FF4800'} },
                    { startValue: 10, endValue: 25, style: { fill: '#FFA200', stroke: '#FFA200'}},
                    { startValue: 25, endValue: 35, style: { fill: '#00B000', stroke: '#00B000'}}],
                pointer: { pointerType: 'default', size: '5%', visible: true, offset: 0 },
                value: 0,
                animationDuration: 0,
            });
        } else {
            Gas.jqxLinearGauge({
                max: 500,
                min: 0,
                height: "80%",
                colorScheme: 'scheme02',
                ticksMajor: { size: '10%', interval: 100 },
                ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                labels: { interval: 100 },
                ranges: [
                    { startValue: 0, endValue: 100, style: { fill: '#FF4800', stroke: '#FF4800'} },
                    { startValue: 100, endValue: 300, style: { fill: '#FFA200', stroke: '#FFA200'}},
                    { startValue: 300, endValue: 500, style: { fill: '#00B000', stroke: '#00B000'}}],
                pointer: { pointerType: 'default', size: '5%', visible: true, offset: 0 },
                value: 0,
                animationDuration: 0,
            });
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

