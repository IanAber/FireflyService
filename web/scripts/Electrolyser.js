var wstimeout;
var elName = "";
var lockSlider = false;
var jsonData;

function smallest(x, y) {
    if ((x > y) && (y > 0)) {
        return y;
    } else {
        return x;
    }
}

function setupPage(name) {
    production = $("#production")
    size = smallest(production.width(), production.height());
    elName = name;
    production.jqxGauge({
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
        caption: {value: 'NL/hr', position: 'bottom', offset: [0, 10], visible: true},
    });
    $("#stackCurrent").jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ticksMinor: {interval: 5, size: '5%'},
        ticksMajor: {interval: 10,size: '9%'},
        labels: {interval:10},
        min: 0,
        max: 75,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Amps', position: 'bottom', offset: [0, 10], visible: true},
    });
    $("#stackVoltage").jqxGauge({
        height: size,
        width: size,
        radius: (size / 2) - 25,
        ticksMinor: {interval: 2, size: '5%'},
        ticksMajor: {interval: 10,size: '9%'},
        labels: {interval:10},
        min: 0,
        max: 50,
        value: 0,
        animationDuration: 500,
        cap: {size: '5%', style: { fill: '#ff0000', stroke: '#00ff00' }, visible: true},
        caption: {value: 'Volts', position: 'bottom', offset: [0, 10], visible: true},
    });
    $("#rate").jqxSlider(
        {
            theme: "energyblue",
            showTickLabels: true,
            tooltip: true,
            mode: "fixed",
            height: 300,
            min: 60,
            max: 100,
            width: 90,
            ticksFrequency: 5,
            value: 0,
            step: 1,
            orientation: "vertical",
            showButtons: false,
            tickLabelFormatFunction: function (value)
            {
                if (value === 60) return value;
                if (value === 100) return value;
                return "";
            }
        }
    );
    rate = $('#rate');
    rate.on('slideEnd', function (event) {
        setRate(event.args.value);
        lockSlider = false;
    });
    rate.on('slideStart', function () {
        lockSlider = true;
    })

    RegisterWebSocket();

    $(window).resize(function () {
        if (this.resizeTO) clearTimeout(this.resizeTO);
        this.resizeTO = setTimeout(function () {
            $(this).trigger('windowResize');
        }, 500);
    });

    $(window).on('windowResize', function () {
        window.location.reload();
    });
}

function RegisterWebSocket() {
    let url = window.origin.replace("http", "ws") + "/ws/electrolyser/" + elName;
    let conn = new WebSocket(url);

    conn.onmessage = function (evt) {
        StartHeartbeat();
        try {
            jsonData = JSON.parse(evt.data);

            if (wstimeout !== 0) {
                clearTimeout(wstimeout);
                $("#connection").hide();
            }

            production = $("#production");
            stackCurrent = $("#stackCurrent");
            stackVoltage = $("#stackVoltage");
            if (jsonData.on) {
                production.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }});
                stackCurrent.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }});
                stackVoltage.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#00e100'}, visible: true }});
            } else {
                production.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }});
                stackCurrent.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }});
                stackVoltage.jqxGauge({ border: { showGradient: true, size: '15%', style: { stroke: '#e10000'}, visible: true }});
            }
            production.jqxGauge({caption:{value:jsonData.h2Flow.toFixed(1) + " NL/hr"}})
            production.val(jsonData.h2Flow);
            stackCurrent.jqxGauge({caption:{value:jsonData.stackCurrent.toFixed(1) + " Amps"}})
            stackCurrent.val(jsonData.stackCurrent);
            stackVoltage.jqxGauge({caption:{value:jsonData.stackVoltage.toFixed(1) + " Volts"}})
            stackVoltage.val(jsonData.stackVoltage);
            $("#model").text(jsonData.model);
            $("#serial").text(jsonData.serial);
            $("#ip").text(jsonData.ip);
            $("#innerh2").text(jsonData.innerH2.toFixed(1));
            $("#outerh2").text(jsonData.outerH2.toFixed(1));
            $("#waterPressure").text(jsonData.waterPressure.toFixed(2));
            $("#temperature").text(jsonData.temp.toFixed(1));
            if (!lockSlider) {
                $("#rate").val(jsonData.rate.toFixed(1));
            }
            $("#maxPressure").text(jsonData.maxPressure.toFixed(0));
            $("#restartPressure").text(jsonData.restartPressure.toFixed(0));
            if (jsonData.errors.length > 0) {
                $("#errors").text(jsonData.errors.codes.join("<br />"));
            } else {
                $("#errors").text("");
            }
            state = $("#state");
            RunButton = $("#Run");
            MaintenanceButton = $("#Maintenance");
            BlowdownButton = $("#Blowdown");
            switch (jsonData.state) {
                case 0 : state.text("Halted");
                break;
                case 1 : state.text("Maintenance mode");
                    setButtonOnOff(RunButton, false);
                    setButtonOnOff(MaintenanceButton, true);
                    setButtonOnOff(BlowdownButton, false);
                    break;
                case 2 : state.text("Idle");
                    setButtonOnOff(RunButton, false);
                    setButtonOnOff(MaintenanceButton, false);
                    setButtonOnOff(BlowdownButton, false);
                    break;
                case 3 : state.text("Steady");
                    setButtonOnOff(RunButton, true);
                    setButtonOnOff(MaintenanceButton, false);
                    setButtonOnOff(BlowdownButton, false);
                    break;
                case 4 : state.text("Stand-By (Max Pressure)");
                    setButtonOnOff(RunButton, true);
                    setButtonOnOff(MaintenanceButton, false);
                    setButtonOnOff(BlowdownButton, false);
                    break;
                case 5 : state.text("Curve");
                    setButtonOnOff(RunButton, false);
                    setButtonOnOff(MaintenanceButton, true);
                    setButtonOnOff(BlowdownButton, false);
                    break;
                case 6 : state.text("Blowdown");
                    setButtonOnOff(RunButton, false);
                    setButtonOnOff(MaintenanceButton, false);
                    setButtonOnOff(BlowdownButton, true);
                break;
                default : state.text("Unknown");
            }
            if (jsonData.dryer == null) {
                $("#DryerDiv").hide();
            } else {
                $("#DryerDiv").show();
                if (jsonData.dryer.temps[0] != null) {
                    $("#temp1").text(jsonData.dryer.temps[0].toFixed(1));
                } else {
                    $("#temp1").text("");
                }
                if (jsonData.dryer.temps[1] != null) {
                    $("#temp2").text(jsonData.dryer.temps[1].toFixed(1));
                } else {
                    $("#temp2").text("");
                }
                if (jsonData.dryer.temps[2] != null) {
                    $("#temp3").text(jsonData.dryer.temps[2].toFixed(1));
                } else {
                    $("#temp3").text("");
                }
                if (jsonData.dryer.temps[3] != null) {
                    $("#temp4").text(jsonData.dryer.temps[3].toFixed(1));
                } else {
                    $("#temp4").text("");
                }
                if (jsonData.dryer.inputPressure != null) {
                    $("#inPressure").text(jsonData.dryer.inputPressure.toFixed(1));
                } else {
                    $("#inPressure").text("");
                }
                if (jsonData.dryer.outputPressure != null) {
                    $("#outPressure").text(jsonData.dryer.outputPressure.toFixed(1));
                } else {
                    $("#outPressure").text("");
                }
            }
        } catch (e) {
            alert(e);
        }
    }
}

function setButtonOnOff(button, on) {
    if (on && button.hasClass("swOff")) {
        button.removeClass("swOff");
        button.addClass("swOn");
        button.removeClass("depressed");
    } else if (!on && button.hasClass( "swOn")) {
        button.removeClass("swOn");
        button.addClass("swOff");
        button.removeClass("depressed");
    }
}

function RunClick() {
    let button = $("#Run");
    button.addClass("depressed");
    let url
    if (button.hasClass("swOn")) {
        url = "/setElectrolyser/Stop/" + elName;
    } else {
        url = "/setElectrolyser/Start/" + elName;
    }
    $.ajax({
        method : "PUT",
        url: url
    });
}

function MaintenanceClick() {
    let button = $("#Maintenance");
    button.addClass("depressed");
    if (button.hasClass("swOn")) {
        url = "/setElectrolyser/StopMaintenance/" + elName;
    } else {
        if (confirm("Entering Maintenance Mode requires that you empty and refill the electrolyser.\nAre you really sure this is what you want to do?") === true) {
            url = "/setElectrolyser/StartMaintenance/" + elName;
        } else {
            return;
        }
    }
    $.ajax({
        method : "PUT",
        url: url
    });
}

function BlowDownClick() {
    let button = $("#BlowDown");
    button.addClass("depressed");
    if (button.hasClass("swOff")) {
        if (confirm("You are about to perform an Electrolyser Blow Down sequence.\nAre you really sure this is what you want to do?") === true) {
            url = "/setElectrolyser/Blowdown/" + elName;
        } else {
            return;
        }
        $.ajax({
            method : "PUT",
            url: url
        });
    }
}

function RescanClick() {
    let button = $("#Rescan");
    if (!button.hasClass("depressed")) {
        if (confirm("You are about to perform an Electrolyser Rescan sequence to try and update the IP address.\nThis should only be performed if you believe the current IP address is incorrect.\nAre you really sure this is what you want to do?") === true) {
            button.addClass("depressed");
            url = "/setElectrolyser/Rescan/" + elName;
        } else {
            return;
        }
        $.ajax({
            method : "PUT",
            url: url
        }).done(function() {
            $("#Rescan").removeClass("depressed");
        })
    }
}

function RefillClick() {
    let button = $("#Refill");
    setButtonOnOff(button, true);
    if (button.hasClass("swOff")) {
        if (confirm("You are about to perform an Electrolyser Refill sequence.\nAre you really sure this is what you want to do?") === true) {
            url = "/setElectrolyser/Refill/" + elName;
        } else {
            return;
        }
        $.ajax({
            method : "PUT",
            url: url
        });
    }
    setTimeout(clearRefill, 5000);
}

function clearRefill() {
    setButtonOnOff($("#Refill"), false)
}

function setRate(rate) {
    if (elName === "") {
        return;
    }
    url = "/setElectrolyser/Production/" + elName + "/" + rate;
    $.ajax({
        method : "PUT",
        url: url
    });
}

function stopDryer() {
    url = "/setDryer/Stop";
    $.ajax({
        method : "PUT",
        url: url
    });
}

function startDryer() {
    url = "/setDryer/Start";
    $.ajax({
        method : "PUT",
        url: url
    });
}

function rebootDryer() {
    url = "/setDryer/Reboot";
    $.ajax({
        method : "PUT",
        url: url
    });
}

function StartHeartbeat() {
    let hb = $("#heartbeat")
    hb.css({width: "20px", height: "20px"})
    hb.animate({width: "15px", height: "15px"})
}
