let jsonData;
let wsConn;

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

function LowBattUp(step) {
    let lb = $("#LowBattDemand");
    val = parseFloat(lb.val()) + step;
    if (val >= 60) {
        val = 60;
    }
    lb.val((val).toFixed(1));
    SendBattLowToFuelCell(lb.val());
}

function LowBattDown(step) {
    let lb = $("#LowBattDemand");
    val = parseFloat(lb.val()) - step;
    if (val <= 30) {
        val = 30;
    }
    lb.val((val).toFixed(1));
    SendBattLowToFuelCell(lb.val());
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
    btn.addClass("ButtonChanging");
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
        $("#ClearFault").addClass("ButtonChanging");
        $.ajax({
            method : "PUT",
            url: url
        });
    } else {
        alert("The fuel cell system must be in Standby to clear faults.");
    }
}

function HeaterClick() {
    let btn = $("#Heater");
    let onOff = btn.hasClass("swOn");
    btn.addClass("ButtonChanging");
    let url = "/setFuelCell/TurnOnHeater";
    if (onOff) {
        url = "/setFuelCell/TurnOffHeater"
    }
    $.ajax({
        method : "PUT",
        url: url
    });
}

function ExhaustClick() {
    if ($("#Enable").hasClass("swOff")) {
        alert("Control is disabled. Click the Enable button to allow control of the fuel cell.");
        return;
    }
    if ($("#FCStatus").text() === "Off") {
        alert("Fuelcel is not on or is not responding.");
        url = "/setFuelCell/ExhaustClose";
    } else {
        let btn = $("#Exhaust");
        btn.addClass("ButtonChanging");
        if (btn.hasClass('swOn')) {
            url = "/setFuelCell/ExhaustClose";
        } else {
            url = "/setFuelCell/ExhaustOpen";
        }
        btn.addClass("ButtonChanging");
    }
    $.ajax({
        method : "PUT",
        url: url
    });
}

function EnableFuelCellClick() {
    let btn = $("#Enable");
    btn.addClass("ButtonChanging");
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
    MonitorWebService();
}

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws/fuelcell";
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
            document.title=jsonData.System;
            $("#version").text(jsonData.Version);
            let sw = $("#Exhaust");
            sw.removeClass("ButtonChanging");
            bOn = (sw.attr('state') === "true");
            if (jsonData.ExhaustOpen) {
                sw.addClass("swOn");
                sw.removeClass("swOff");
            } else {
                sw.addClass("swOff");
                sw.removeClass("swOn");
            }

            let en = $("#Enable");
            en.removeClass("ButtonChanging");
            if (jsonData.Enable) {
                en.addClass("swOn");
                en.removeClass("swOff");
            } else {
                en.addClass("swOff");
                en.removeClass("swOn")
            }

            let cf = $("#ClearFault");
            if (!jsonData.ClearFaultsActive) {
                cf.removeClass("ButtonChanging");
            }

            let htr = $("#Heater");
            htr.removeClass("ButtonChanging")
            if (!jsonData.HeaterOn) {
                htr.addClass("swOff");
                htr.removeClass("swOn");
            } else {
                htr.addClass("swOn");
                htr.removeClass("swOff");
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

            let onOff = $("#SwitchOnOff");
            let on = false;
            let onOffBtnEnable = true;

            switch (jsonData.RunStatus) {
                case "Standby" :
                    on = false;
                    onOffBtnEnable = true;
                    break;
                case "Hydrogen leak check" :
                    on = true;
                    onOffBtnEnable = false;
                    break;
                case "Hydrogen intake" :
                    on = true;
                    onOffBtnEnable = false;
                    break;
                case "Start" :
                    on = true;
                    onOffBtnEnable = true;
                    break;
                case "AirPurge" :
                    on = false;
                    onOffBtnEnable = false;
                    break;
                case "manual" :
                    on = false;
                    onOffBtnEnable = false;
                    break;
                case "emergency stop" :
                    on = false;
                    onOffBtnEnable = false;
                    break;
                case "fault" :
                    on = false;
                    onOffBtnEnable = false;
                    break;
                case "shutdown" :
                    on = false;
                    onOffBtnEnable = false;
                    break;
                default :
            }
            onOff.removeClass("ButtonChanging");
            if (on) {
                onOff.addClass("swOn");
                onOff.removeClass("swOff");
            } else {
                onOff.addClass("swOff");
                onOff.removeClass("swOn");
            }

            if (onOffBtnEnable) {
                onOff.prop("disabled", false);
            } else {
                onOff.prop("disabled", true);
            }
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
    if ((CurrentsVals[0] !== CurrentsNewVals[0]) || (CurrentsVals[1] !== CurrentsNewVals[1])) {
        Currents.val(CurrentsNewVals);
    }

    PowersNewVals = [jsonData.BMSTargetPower, jsonData.BMSCurrentPower];
    Powers = $("#fcPower");
    PowersVals = Powers.val();
    if ((PowersVals[0] !== PowersNewVals[0]) || (PowersVals[1] !== PowersNewVals[1])) {
        Powers.val(PowersNewVals);
    }

    $("#fcStackPower").val(jsonData.StackPower);
    $("#fcStackVolts").val(jsonData.StackVolts);
    $("#fcStackCurrent").val(jsonData.StackCurrent);
    $("#fcWaterPumpSpeed").val(jsonData.WaterPumpSpeed);
    $("#fcCoolingFanSpeed").val(jsonData.CoolingFanSpeed);

    try {
        showFuelCellAlarms(jsonData, $("#fcAlarms"));
    } catch(e) {
        console.log(e);
    }
}

function setUpFuelCellGauges() {
    let power = $('#fcPower');
    let size = smallest(power.width(), power.height());
    power.jqxBarGauge({
        width: size,
        height: size,
        values: [0.0, 0.0],
        min: 0,
        max: 12,
        animationDuration: 0,
        startAngle: 265,
        endAngle: 275,
        title:	{
            text: 'Power',
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
                    case 0 : return("Setpoint = " + value + 'kW');
                    default : return ("Actual = " + value + 'kW');
                }
            }}
    });

    let pressures = $("#fcPressures");
    size = smallest(pressures.width(), pressures.height());
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
