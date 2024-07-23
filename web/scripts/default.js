var jsonData;
let elCount = 0;
let maxFlow = 0;

// function smallest(v1, v2) {
//     return (v1 < v2) ? v1 : v2;
// }

let gasUnits = { units: '',
                 displayUnits: '',
                 capacity: 0};

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws";
    let conn = new WebSocket(url);
    let diameter = 0;

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

            let gas = $("#gas");
            let gasWidth = gas.width();
            let gasLeft = gas.offset().left;
            let FuelCell = $("#fc");
            if (jsonData.PanFuelCellStatus !== null) {
                $("FuelCell").show();
                FuelCell.val(jsonData.PanFuelCellStatus.StackPower);
                if (jsonData.PanFuelCellStatus.StackPower > 0) {
                    let usingH2Div = $("#usingH2Div");
                    let rightPos = FuelCell.offset().left + (FuelCell.width() *0.1);
                    let topPos = FuelCell.offset().top + (FuelCell.height() / 2) - 20;
                    let leftPos = gasLeft + (gasWidth * 0.8);
                    // Set the width to no more than the width of the graphic
                    let usingWidth = smallest(rightPos - leftPos, $("#usingH2").width());
                    let usingCss = {
                        left: leftPos + "px",
                        top: topPos + "px",
                        width: usingWidth + "px"
                    };
                    usingH2Div.css(usingCss);

                    usingH2Div.show();
                } else {
                    $("#usingH2Div").hide();
                }
                diameter = smallest(FuelCell.jqxGauge('height'), FuelCell.jqxGauge('width'));
                showFuelCellAlarms(jsonData.PanFuelCellStatus, $("#fcAlarms"));
            } else {
                FuelCell.hide();
            }

            let flow = 0;
            let EL = $("#el");
            if (elCount !== jsonData.Electrolysers.length) {
                elCount = jsonData.Electrolysers.length;
                maxFlow = Math.round(elCount * ((5250 / 990) + 0.5)) / 10;
                EL.jqxGauge({
                    min: 0,
                    max: maxFlow,
                    ticksMinor: {interval: maxFlow / 20, size: '5%'},
                    ticksMajor: {interval: maxFlow / 10,size: '9%'},
                    labels: {interval:maxFlow/ 10, position: "far", formatValue: function(value){return (value * 1).toFixed(2)} },
                    value: 0,
                    animationDuration: 500,
                    cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
                    caption: {value: 'kW', position: 'bottom', offset: [0, 10], visible: true},
                });
                if (diameter === 0) {
                    diameter = EL.jqxGauge('height');
                } else {
                    diameter = smallest( diameter, EL.jqxGauge('height'));
                }
                diameter = smallest( diameter, EL.jqxGauge('width'));
                EL.css({"width":diameter});
                EL.jqxGauge({width: diameter, height: diameter});
                FuelCell.css({width: diameter});
                FuelCell.jqxGauge({width: diameter, height: diameter});
            } else {
                diameter = EL.jqxGauge('width');
            }
            jsonData.Electrolysers.forEach((currentElement) => {flow += currentElement.h2Flow});
            EL.val(flow / 990);

            let storingH2Div = $("#storingH2Div");
            if (flow > 0) {
                let leftPos = EL.offset().left + (EL.width() *0.9);
                let topPos = EL.offset().top + (EL.height() / 2) - 20;
                let rightPos = gasLeft + (gasWidth / 2);


                // Set the width to no more than the width of the graphic
                let storingWidth = smallest(rightPos - leftPos, $("#storingH2").width());
                let storingCss = {
                    left: leftPos + "px",
                    top: topPos + "px",
                    width: storingWidth + "px"
                };
                storingH2Div.css(storingCss);
                storingH2Div.show();
            } else {
                storingH2Div.hide();
            }

            gasUnits.capacity = jsonData.SystemSettings.gasCapacity;
            updatePressure(jsonData.Analog.Inputs[jsonData.SystemSettings.gasInput].Value, jsonData.SystemSettings.gasUnits, jsonData.SystemSettings.gasDisplayUnits, jsonData.SystemSettings.gasCapacity, jsonData.Analog.gasTemperature);
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

function buildURLForTimes(start, end) {
    if (end - start > 86400000) {   // More than 24 hours?
        start = new Date(start.getFullYear(), start.getMonth(), start.getDate());
        end = new Date(end.getFullYear(), end.getMonth(), end.getDate());
    } else {
        start = new Date(start.getFullYear(), start.getMonth(), start.getDate(), start.getHours());
        end = new Date(end.getFullYear(), end.getMonth(), end.getDate(), end.getHours());
    }
    $("#startAt").jqxDateTimeInput('setDate', start );
    $("#endAt").jqxDateTimeInput('setDate', end );
    if ((end - start) > 86400) {

    }
    url = encodeURI("../Hydrogen/Data/?start="
        + start.getUTCFullYear() + "-" + (start.getUTCMonth() + 1) + "-" + start.getUTCDate() + " " + start.getUTCHours() + ":" + start.getUTCMinutes()
        + "&end=" + end.getUTCFullYear() + "-" + (end.getUTCMonth() + 1) + "-" +  end.getUTCDate() + " " + end.getUTCHours() + ":" + end.getUTCMinutes());
    return url;
}

function buildChart() {
    let Settings = {
        title: "Hydrogen (kg)",
        description: "Hydrogen consumption and production",
        enableAnimations: true,
        animationDuration: 1000,
        enableAxisTextAnimation: true,
        showLegend: true,
        padding: { left: 5, top: 5, right: 5, bottom: 5 },
        titlePadding: { left: 90, top: 0, right: 0, bottom: 10 },
        xAxis: {
            dataField: 'logged',
            type: 'date',
            showGridLines: false,
            textRotationAngle: 270,
            formatFunction: xAxisFormatFunction,
            minValue: start,
            maxValue: end,
            baseUnit: 'hour',
            unitInterval: 1,   // 1 hour
            labels: {
                step: 1,
            }
        },
        colorScheme: 'scheme01',
        seriesGroups: [{
            type: 'column',
            valueAxis: {
                gridLines: {
                    visible: false,
                },
                labels: {
                    formatSettings: {
                        decimalPlaces: 2,
                    },
                    visible: true,
                },
                description: 'H2',
            },
            series: [{
                dataField: 'increase',
                displayText: 'Produced'

            },{
                dataField: 'decrease',
                displayText: 'Used'
            }]
        }]
    }
    setupChart(Settings);
}

function postUpdate(data) {
    let span = data[data.length - 1].logged - data[0].logged;
    let Chart = $('#ChartContainer');
    let xAxis = Chart.jqxChart('xAxis');
    if (span > 86400000) {
        xAxis.baseUnit = 'day';
    } else {
        xAxis.baseUnit = 'hour';
    }
    xAxis.unitInterval = 1;
    Chart.jqxChart('update');

    let totalH2Produced = 0;

    for (i = 0; i < data.length; i++) {
        totalH2Produced += data[i].increase;
    }
    let co2Saved = totalH2Produced * 4.5

    if (gasUnits.units === "bar") {
        units = "kg";
    } else {
        units = "lbs";
        co2Saved = co2Saved * 2.2;
    }
    $("#co2").html(`<h2>${co2Saved.toFixed(2)}${units} of CO<sub>2</sub> Saved</h2>`);
}

function setupPage() {
    let FuelCell = $("#fc");
    FuelCell.jqxGauge({
        ticksMinor: {interval: 500, size: '5%'},
        ticksMajor: {interval: 1000,size: '9%'},
        labels: {interval:5000, position: "far", formatValue(value){return (value / 1000).toFixed(2)}},
        min: 0,
        max: 15000,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Fuel Cell (kW)', position: 'bottom', offset: [0, 10], visible: true},
    });
//    buildChart();
    RegisterWebSocket();
}

function updatePressure(pressure, units, displayUnits, capacity) {
    let Gas = $("#gas");
    if ((gasUnits.units !== units) || (gasUnits.displayUnits !== displayUnits)) {
        gasUnits.units = units;
        gasUnits.displayUnits = displayUnits;
        buildChart();
    }
    if (Gas.children().length < 1) {
        let gasTitle = $("#gasTitle");
        switch (displayUnits)  {
            case "bar" :
                Gas.jqxLinearGauge({
                    max: 35,
                    min: 0,
                    height: "80%",
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
                    max: 510,
                    min: 0,
                    height: "80%",
                    colorScheme: 'scheme02',
                    ticksPosition: 'far',
                    ticksOffset: [30, 15],
                    ticksMajor: { size: '10%', interval: 100 },
                    ticksMinor: { size: '5%', interval: 50, style: { 'stroke-width': 1, stroke: '#aaaaaa'} },
                    labels: { interval: 100, position: 'far' },
                    ranges: [
                        { startValue: 0, endValue: 145, style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: 145, endValue: 406, style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: 406, endValue: 510, style: { fill: '#00B000', stroke: '#00B000'}}],
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
                        { startValue: 0, endValue: Math.round(capacity * 0.29), style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: Math.round(capacity * 0.29), endValue: Math.round(capacity * 0.8), style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: Math.round(capacity * 0.8), endValue: capacity, style: { fill: '#00B000', stroke: '#00B000'}}],
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
                        { startValue: 0, endValue: Math.round(cuft * 0.29), style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: Math.round(cuft * 0.29), endValue: Math.round(cuft * 0.8) , style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: Math.round(cuft * 0.8), endValue: cuft, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("H2 Volume (cubic feet)");
                break;
            case "kWhr" :
                // using 859l/kWhr
                let kWhr = Math.round(((capacity * 35) / 859) - ((capacity * 10) / 859)); // Conversion from litres to kWhr
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
                        { startValue: 0, endValue: Math.round(kWhr * 0.2), style: { fill: '#FF4800', stroke: '#FF4800'} },
                        { startValue: Math.round(kWhr * 0.2), endValue: Math.round(kWhr * 0.8), style: { fill: '#FFA200', stroke: '#FFA200'}},
                        { startValue: Math.round(kWhr * 0.8), endValue: kWhr, style: { fill: '#00B000', stroke: '#00B000'}}],
                    pointer: { pointerType: 'rectangle', size: '15%', visible: true, offset: 0 },
                    value: 0,
                    animationDuration: 0,
                });
                gasTitle.text("Energy Available (kWhr)");
                break;
        }
    }
    if (units === "bar") {
        switch (displayUnits) { // capacity is in litres
            case "litres" : pressure = ((pressure / 35) * capacity).toFixed(0); // Bar -> litres
                break;
            case "cuft" : pressure = ((pressure / 35) * capacity * 0.03535).toFixed(1); // Bar -> cubic feet
                break;
            case "kWhr" : pressure = ((capacity * pressure) / 859) - ((capacity * 10) / 859); // Bar -> kwHr (minimum pressure = 10Bar)
                if (pressure < 0) {
                    pressure = 0;
                }
                break;
            case "psi" : pressure = (pressure * 14.5).toFixed(0)
        }
    } else if (units === "psi") {
        switch (displayUnits) { // capacity is in litres
            case "litres" : pressure = ((pressure / 507.6) / capacity).toFixed(0); // psi -> litres
                break;
            case "cuft" : pressure = ((pressure / 507.6) * capacity * 0.03535).toFixed(1); // psi -> cubic feet
                break;
            case "kWhr" : pressure = (pressure * 0.0689).toFixed(1); // Convert to Bar
                pressure = ((capacity * pressure) / 859) - ((capacity * 10) / 859); // Bar -> kwHr (minimum pressure = 10Bar)
                if (pressure < 0) {
                    pressure = 0;
                }
                break;
            // case "kWhr" : pressure = (((pressure / 507.6) * capacity) / 990).toFixed(1); // psi -> kwHr
            //     break;
            case "bar" : pressure = (pressure * 0.0689).toFixed(1); // psi -> bar
        }
    }
    Gas.val(pressure);

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

