var wstimeout;

function smallest(x, y) {
    if ((x > y) && (y > 0)) {
        return y;
    } else {
        return x;
    }
}

function PowerDown(step) {
    let pd = $("#PowerDemand");
    val = parseFloat(pd.val()) - step;
    if (val < 0) {
        val = 0;
    }
    pd.val((val).toFixed(1));
    SendPowerToFuelCell(pd.val());
}

function PowerUp(step) {
    let pd = $("#PowerDemand");
    val = parseFloat(pd.val()) + step;
    if (val >= 10) {
        val = 10;
    }
    pd.val((val).toFixed(1))
    SendPowerToFuelCell(pd.val());
}

function HighBattUp(step) {
    let hb = $("#HighBattDemand");
    val = parseFloat(hb.val()) + step;
    if (val >= 70) {
        val = 70;
    }
    hb.val((val).toFixed(1));
    SendBattHighToFuelCell(hb.val());
}

function HighBattDown(step) {
    let hb = $("#HighBattDemand");
    val = parseFloat(hb.val()) - step;
    if (val <= 35) {
        val = 35;
    }
    hb.val((val).toFixed(1));
    SendBattHighToFuelCell(hb.val());
}

function LowBattUp() {
    let ld = $("#LowBattDemand");
    val = parseFloat(ld.val());
    if (val >= 70) {
        return;
    }
    ld.val((val + 0.1).toFixed(1));
    SendBattLowToFuelCell(hb.val());
}

function LowBattDown() {
    let ld = $("#LowBattDemand");
    val = parseFloat(ld.val());
    if (val <= 35) {
        return;
    }
    ld.val((val - 0.1).toFixed(1));
    SendBattLowToFuelCell(hb.val());
}

function SendPowerToFuelCell(power) {
    let url = "/setFuelCell/TargetPower/" + power
    $.ajax({
        method : "PUT",
        url: url
    });
}

function SendBattHighToFuelCell(volts) {
    let url = "/setFuelCell/TargetBattHigh/" + volts
    $.ajax({
        method : "PUT",
        url: url
    });
}

function SendBattLowToFuelCell(volts) {
    let url = "/setFuelCell/TargetBattLow/" + volts
    $.ajax({
        method : "PUT",
        url: url
    });
}

function RunFuelCellClick() {
    if ($("#Enable").hasClass("swOff")) {
        alert("Control is disabled. Click the Enable button to allow control of the fuel cell.");
        return;
    }
    let btn = $("#SwitchOnOff");
    let onOff = btn.hasClass("swOn");
    btn.addClass("depressed");
    if (onOff) {
        url = "/setFuelCell/Stop";
    } else {
        url = "/setFuelCell/Start";
    }
    $.ajax({
        method : "PUT",
        url: url
    });
}

function ClearFaultClick() {
    if ($("#FCStatus").text() === "Standby") {
        url = "/setFuelCell/ResetFault";
        $("#ClearFault").addClass("depressed");
        $.ajax({
            method : "PUT",
            url: url
        });
    } else {
        alert("The fuel cell system must be in Standby to clear faults.");
    }
}

function ExhaustClick() {
    if ($("#Enable").hasClass("swOff")) {
        alert("Control is disabled. Click the Enable button to allow control of the fuel cell.");
        return;
    }
    let btn = $("#Exhaust");
    btn.addClass("depressed");
    if (btn.hasClass('swOn')) {
        url = "/setFuelCell/ExhaustClose";
    } else {
        url = "/setFuelCell/ExhaustOpen";
    }
    btn.addClass("depressed");
    $.ajax({
        method : "PUT",
        url: url
    });
}

function EnableFuelCellClick() {
    let btn = $("#Enable");
    btn.addClass("depressed");
    if (btn.hasClass("swOn")) {
        url = "/setFuelCell/Disable";
    } else {
        url = "/setFuelCell/Enable";
    }
    $.ajax({
        method : "PUT",
        url: url
    });
}

function UpdateFuelCell() {
    $("#settingsForm").submit();
}

function setupPage() {
    setUpFuelCellGauges();

    $('#PowerDemand').on('input', function() {
        val = $(this).val();
        if (val >= 0 && val <= 10) {
            SendPowerToFuelCell(val) // get the current value of the input field.
        }
    });
    $('#HighBattDemand').on('input', function() {
        val = $(this).val();
        if (val >= 35 && val <= 70) {
            SendBattHighToFuelCell(val) // get the current value of the input field.
        }
    });
    $('#LowBattDemand').on('input', function() {
        val = $(this).val();
        if (val >= 35 && val <= 70) {
            SendBattLowToFuelCell(val) // get the current value of the input field.
        }
    });
    $(window).resize(function () {
        if (this.resizeTO) clearTimeout(this.resizeTO);
        this.resizeTO = setTimeout(function () {
            $(this).trigger('windowResize');
        }, 500);
    });
    RegisterWebSocket();
}

var jsonData;

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws/fuelcell";
    let conn = new WebSocket(url);

    conn.onmessage = function (evt) {
        // Restart the timeout timer
        StartHeartbeat();

        try {
            jsonData = JSON.parse(evt.data);
            $("#system").text(jsonData.System);
            $("#version").text(jsonData.Version);
            let sw = $("#Exhaust");
            sw.removeClass("depressed");
            bOn = (sw.attr('state') === "true");
            if (jsonData.ExhaustOpen) {
                sw.addClass("swOn");
                sw.removeClass("swOff");
            } else {
                sw.addClass("swOff");
                sw.removeClass("swOn");
            }

            let onOff = $("#SwitchOnOff");
            onOff.removeClass("depressed");
            if (jsonData.Start) {
                onOff.addClass("swOn");
                onOff.removeClass("swOff");
            } else {
                onOff.addClass("swOff");
                onOff.removeClass("swOn");
            }

            let en = $("#Enable");
            en.removeClass("depressed");
            if (jsonData.Enable) {
                en.addClass("swOn");
                en.removeClass("swOff");
            } else {
                en.addClass("swOff");
                en.removeClass("swOn")
            }

            let cf = $("#ClearFault");
            if (!jsonData.ClearFaultsActive) {
                cf.removeClass("depressed");
            }

            highBattDemand = $("#HighBattDemand");
            if (!highBattDemand.is(":focus")) {
                highBattDemand.val(jsonData.BMSTargetHigh.toFixed(1));
            }
            lowBattDemand = $("#LowBattDemand");
            if (!lowBattDemand.is(":focus")) {
                lowBattDemand.val(jsonData.BMSTargetLow.toFixed(1));
            }
            powerDemand = $("#PowerDemand");
            if (!powerDemand.is(":focus")) {
                powerDemand.val(jsonData.BMSTargetPower.toFixed(1));
            }
            $("#FCStatus").text(jsonData.RunStatus);
            $("#FCDCOutputStatus").text(jsonData.DCOutputStatus);
            UpdateGauges(jsonData);


        } catch (e) {
            alert(e);
        }
    }
}

function UpdateGauges(jsonData) {
    PressuresNewVals = [jsonData.H2Pressure,
        jsonData.AirPressure,
        jsonData.CoolantPressure,
        jsonData.H2AirPressureDiff];
    Pressures = $("#fcPressures");
    PressuresVals = Pressures.val();
    if ((PressuresVals[0] !== PressuresNewVals[0]) || (PressuresVals[1] !== PressuresNewVals[1]) ||
        (PressuresVals[2] !== PressuresNewVals[2]) || (PressuresVals[3] !== PressuresNewVals[3]))
    {
        Pressures.val(PressuresNewVals);
    }

    TemperaturesNewVals = [jsonData.CoolantInletTemp,
        jsonData.CoolantOutletTemp,
        jsonData.AirTemp,
        jsonData.AmbientTemp];
    Temperatures = $("#fcTemperatures");
    TemperaturesVals = Temperatures.val();
    if ((TemperaturesVals[0] !== TemperaturesNewVals[0]) || (TemperaturesVals[1] !== TemperaturesNewVals[1]) ||
        (TemperaturesVals[2] !== TemperaturesNewVals[2]) || (TemperaturesVals[3] !== TemperaturesNewVals[3]))
    {
        Temperatures.val(TemperaturesNewVals);
    }

    VoltagesNewVals = [jsonData.DCInVolts,
        jsonData.DCOutVolts];
    Voltages = $("#fcVoltages");
    VoltagesVals = Voltages.val();
    if ((VoltagesVals[0] !== VoltagesNewVals[0]) || (VoltagesVals[1] !== VoltagesNewVals[1])) {
        Voltages.val(VoltagesNewVals);
    }

    CurrentsNewVals = [jsonData.DCInAmps, jsonData.DCOutAmps];
    Currents = $("#fcCurrent");
    CurrentsVals = Currents.val();
    if ((CurrentsVals[0] !== CurrentsNewVals[0]) || (CurrentsVals[1] !== CurrentsNewVals[1]))
        Currents.val(CurrentsNewVals);

    if (jsonData.InsulationResistance === 65535) {
        $("#InsulationDiv").hide();
    } else {
        $('#fcInsulation').val(jsonData.InsulationResistance);
        $('#fcInsulationStatus').text(jsonData.InsulationStatus);
        $("#fcInsulationFault").text(jsonData.InsulationFault);
    }
    $("#fcStackPower").val(jsonData.StackPower);
    $("#fcStackVolts").val(jsonData.StackVolts);
    $("#fcStackCurrent").val(jsonData.StackCurrent);
    $("#fcWaterPumpSpeed").val(jsonData.WaterPumpSpeed);
    $("#fcCoolingFanSpeed").val(jsonData.CoolingFanSpeed);
    var alarmText;
    let alarmDiv = $("#fcAlarms");
    if (jsonData.Alarms.length > 0) {
        alarmText = '<span class="alarm">';
        alarmText += jsonData.Alarms.join('</span><br /><span class="alarm">')
        alarmText += '</span>'
    } else {
        alarmText = "";
    }
    if (jsonData.DCOutputStatus === "fault") {
        alarmText += '<span class="alarm">' + jsonData.DCOutputFault + '</span><br />'
    }
    if (alarmText !== "") {
        alarmDiv.html(alarmText);
        alarmDiv.show();
    } else {
        alarmDiv.hide();
    }
}

function setUpFuelCellGauges() {
    let pressures = $("#fcPressures");
    let size = smallest(pressures.width(), pressures.height());
    pressures.jqxBarGauge({
        width: size,
        height: size,
        values: [0.0, 0.0, 0.0, 0.0],
        min: 0,
        max: 150,
        animationDuration: 0,
        startAngle: 265,
        endAngle: 275,
        title:	{
            text: 'Pressures',
            font: { size: 12, color: 'black', weight: 'bold', family:"Segoi-UI"},
            margin: { top: 0, bottom: 2, left: 0, right: 0},
            verticalAlignment: 'bottom'
        },
        labels: {
            font: {size: 8,},
            precision: 1,
        },
        colorScheme: 'customColors',
        customColorScheme: { name: 'customColors', colors: ['#000066', '#006600', '#660000', '#006666'] },
        tooltip: {visible: true,
            precision: 1,
            formatFunction: function (value, index){
                switch(index)
                {
                    case 0 : return("H2 = " + value + 'mbar');
                    case 1 : return ("Air = " + value + 'mbar');
                    case 2 : return ("Coolant = " + value + 'mbar');
                    default : return ("H2/air = " + value + 'mbar');
                }
            }}
    });
    let temperatures= $("#fcTemperatures");
    size = smallest(temperatures.width(), temperatures.height());
    temperatures.jqxBarGauge({
        width: size,
        height: size,
        values: [0.0, 0.0, 0.0, 0.0],
        min: 5,
        max: 90,
        animationDuration: 0,
        startAngle: 265,
        endAngle: 275,
        title:	{
            text: 'Temperatures',
            font: { size: 12, color: 'black', weight: 'bold', family:"Segoi-UI"},
            margin: { top: 0, bottom: 2, left: 0, right: 0},
            verticalAlignment: 'bottom'
        },
        labels: {
            font: {size: 8,},
            precision: 1,
        },
        colorScheme: 'customColors',
        customColorScheme: { name: 'customColors', colors: ['#000066', '#006600', '#660000', '#006666'] },
        tooltip: {visible: true,
            precision: 1,
            formatFunction: function (value, index){
                switch(index)
                {
                    case 0 : return("Inlet = " + value + '&#8451;');
                    case 1 : return ("Outlet = " + value + '&#8451;');
                    case 2 : return ("Air = " + value + '&#8451;');
                    default : return ("Ambient = " + value + '&#8451;');
                }
            }}
    });
    let voltages = $('#fcVoltages');
    size = smallest(voltages.width(), voltages.height());
    voltages.jqxBarGauge({
        width: size,
        height: size,
        values: [0.0, 0.0],
        min: 0,
        max: 85,
        animationDuration: 0,
        startAngle: 265,
        endAngle: 275,
        title:	{
            text: 'Voltages',
            font: { size: 12, color: 'black', weight: 'bold', family:"Segoi-UI"},
            margin: { top: 0, bottom: 2, left: 0, right: 0},
            verticalAlignment: 'bottom'
        },
        labels: {
            font: {size: 8,},
            precision: 1,
        },
        colorScheme: 'customColors',
        customColorScheme: { name: 'customColors', colors: ['#000066', '#006600'] },
        tooltip: {visible: true,
            precision: 1,
            formatFunction: function (value, index){
                switch(index)
                {
                    case 0 : return("In = " + value + 'V');
                    default : return ("Out = " + value + 'V');
                }
            }}
    });
    let currents = $('#fcCurrent');
    size = smallest(currents.width(), currents.height());
    currents.jqxBarGauge({
        width: size,
        height: size,
        values: [0.0, 0.0],
        min: 0,
        max: 400,
        animationDuration: 0,
        startAngle: 265,
        endAngle: 275,
        title:	{
            text: 'Current',
            font: { size: 12, color: 'black', weight: 'bold', family:"Segoi-UI"},
            margin: { top: 0, bottom: 2, left: 0, right: 0},
            verticalAlignment: 'bottom'
        },
        labels: {
            font: {size: 8,},
            precision: 1,
        },
        colorScheme: 'customColors',
        customColorScheme: { name: 'customColors', colors: ['#000066', '#006600'] },
        tooltip: {visible: true,
            precision: 1,
            formatFunction: function (value, index){
                switch(index)
                {
                    case 0 : return("In = " + value + 'A');
                    default : return ("Out = " + value + 'A');
                }
            }}
    });
    insulation = $('#fcInsulation');
    size = smallest(insulation.width(), insulation.height());
    insulation.jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ticksMinor: {interval: 50, size: '5%'},
        ticksMajor: {interval: 250,size: '9%'},
        labels: {interval:250},
        min: 0,
        max: 1000,
        value: 0,
        ranges: [
            {startValue: 0, endValue: 100, style: {fill: 'RED', stroke: 'RED'}, startWidth: 9, endWidth: 7},
            {startValue: 100, endValue: 500, style: {fill: 'ORANGE', stroke: 'ORANGE'}, startWidth: 7, endWidth: 5},
            {startValue: 500, endValue: 5000, style: {fill: 'GREEN', stroke: 'GREEN'}, startWidth: 5, endWidth: 2}
        ],
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Insulation', position: 'bottom', offset: [0, 10], visible: true},
    });
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
    stackCurrent = $('#fcStackCurrent');
    size = smallest(stackCurrent.width(), stackCurrent.height());
    stackCurrent.jqxGauge({
        height: size,
        width: size,
        radius: (size/ 2) - 25,
        ticksMinor: {interval: 5, size: '5%'},
        ticksMajor: {interval: 50,size: '9%'},
        labels: {interval:50},
        min: 0,
        max: 400,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Stack Current', position: 'bottom', offset: [0, 10], visible: true},
    });
    stackVolts = $('#fcStackVolts');
    size = smallest(stackVolts.width(), stackVolts.height());
    stackVolts.jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ranges: [
            {startValue: 0, endValue: 30, style: {fill: 'RED', stroke: 'RED'}, startWidth: 9, endWidth: 5},
            {startValue: 30, endValue: 65, style: {fill: 'GREEN', stroke: 'GREEN'}, startWidth: 5, endWidth: 5},
            {startValue: 65, endValue: 80, style: {fill: 'RED', stroke: 'RED'}, startWidth: 5, endWidth: 9}
        ],
        ticksMinor: {interval: 5, size: '5%'},
        ticksMajor: {interval: 20,size: '9%'},
        labels: {interval:10},
        min: 0,
        max: 80,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Stack Voltage', position: 'bottom', offset: [0, 10], visible: true},
    });
    pumpSpeed = $('#fcWaterPumpSpeed');
    size = smallest(pumpSpeed.width(), pumpSpeed.height());
    pumpSpeed.jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ticksMinor: {interval: 500, size: '5%'},
        ticksMajor: {interval: 1000,size: '9%'},
        labels: {interval:1000},
        min: 0,
        max: 6000,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Water Pump Speed', position: 'bottom', offset: [0, 10], visible: true},
    });
    fanSpeed = $('#fcCoolingFanSpeed');
    size = smallest(fanSpeed.width(), fanSpeed.height());
    fanSpeed.jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ticksMinor: {interval: 500, size: '5%'},
        ticksMajor: {interval: 1000,size: '9%'},
        labels: {interval:1000},
        min: 0,
        max: 6000,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Cooling Fan Speed', position: 'bottom', offset: [0, 10], visible: true},
    });
}

function WebSocketTimedOut() {
    RegisterWebSocket();
}

function StartHeartbeat() {
    hb = $("#heartbeat");
    hb.css({width: "20px", height: "20px"})
    hb.animate({width: "15px", height: "15px"})
}
