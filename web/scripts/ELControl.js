// noinspection CommaExpressionJS
const ELDIV = `<div class="centered">
    <div class="control" id="el{{id}}" ondblclick="openElectrolyser('{{name}}', 'el{{id}}')" oncontextmenu="openElectrolyserMain('{{name}}')">
    </div>
    <div class="control" id="elControls{{id}}">
        <div class="control">
            <label class="parameters" for="elRate{{id}}">Production Rate</label><br \>
            <div style="margin:auto" id="elRate{{id}}"></div>
        </div>
        <div class="control" style="display: grid; grid-template-columns: 40% 20% 40%" >
            <div>
                <div class="control">
                    <img id="ELPower{{id}}" class="swOff" src="images/power-off.png" alt="Enable" onclick="PowerClick({{id}}, 'ELPower{{id}}')" />
                    <label for="ELPower{{id}}">Power</label></td>
                </div>
                <div class="control">
                    <img id="ELRun{{id}}" class="swOff" src="images/power-off.png" alt="Enable" onclick='RunClick({{id}}, "ELRun{{id}}", "{{name}}")' />
                    <label for="ELRun{{id}}">Run</label></td>
                </div>
            </div>
            <div>
                <div class="control">
                    <span id="ELStatus{{id}}">Off</span>
                </div>
                <div>
                    <span id="ELDisabled{{id}}" style="color:orange; display:none">DISABLED</span>
                </div>
            </div>
            <div class="control">
                <img id="ELReboot{{id}}" class="swOff" src="images/power-off.png" alt="Enable" onclick='RebootClick({{id}}, "ELReboot{{id}}", "{{name}}")' />
                <label for="ELReboot{{id}}">Reboot</label></td>
            </div>
        </div>
        <div class="warnings" id="warnings{{id}}">
            <span class="warnings" id="ELWarnings{{id}}"></span>
        </div>
        <div class="errors" id="errors{{id}}">
            <span class="errors" id="ELErrors{{id}}"></span>
        </div>
    </div>
</div>`

var lockSliders = false;

function addElectrolyser(id, name) {

    systems = $("#systems");
    sCode = ELDIV.replace(/{{id}}/g, id).replace(/{{name}}/g, name);
    systems.append(sCode)
    //    jQuery('#systems').append(sCode);
}

function updateElectrolyser(currentElement, index) {
    let EL = "#el" + index;
    if ($(EL).length === 0) {
        addElectrolyser(index, currentElement.name);
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
            caption: {value: currentElement.name + ' ' +  currentElement.h2Flow.toFixed(0) + ' NL/hr', position: 'bottom', offset: [0, 10], visible: true},
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
    let elect = $(("#el" + index));
    if (currentElement.on && currentElement.state !== 0) {
        elect.attr('on', "1");
        elect.jqxGauge({
            border: {
                showGradient: true,
                size: '15%',
                style: {
                    stroke: '#00e100'
                },
                visible: true
            },
            width: "100%",
            caption: {
                value: currentElement.name + ' ' +  currentElement.h2Flow.toFixed(0) + ' NL/hr',
                position: 'bottom',
                offset: [0, 10],
                visible: true
            }
        });
    } else {
        elect.attr('on', "0");
        elect.jqxGauge({
            border: {
                showGradient: true,
                size: '15%',
                style: {
                    stroke: '#e10000'
                },
                visible: true
            },
            width: "100%",
            caption: {
                value: currentElement.name + ' 0 NL/hr',
                position: 'bottom',
                offset: [0, 10],
                visible: true
            }
        });
    }
    if (elect.width() > elect.height()) {
        elect.width(elect.height());
    }
//    el.height(el.width());
    setOnOffButton("ELPower"+index, currentElement.on);
    elect.val(currentElement.h2Flow);
    if ((currentElement.state === 3) || (currentElement.state === 4)) {
        setOnOffButton("ELRun" + index, true)
    } else {
        setOnOffButton("ELRun" + index, false)
    }
    let disabled = $("#ELDisabled" + index);
    if (currentElement.enabled) {
        disabled.hide();
    } else {
        disabled.show();
    }
    $("#ELReboot" + index).removeClass("ButtonChanging");
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
    rate = $(('#elRate' + index));
    if (!lockSliders) {
        rate.val(currentElement.rate);
    }
    if (currentElement.on) {
        rate.jqxSlider({ disabled:false });
    } else {
        rate.jqxSlider({ disabled:true });
    }
    let warningsDiv = $("#warnings" + index);
    let warningsSpan = $("#ELWarnings" + index);
    if (currentElement.warnings != null) {
        warningsDiv.show();
        warningsSpan.text(currentElement.warnings.join("<br />"));
    } else {
        warningsDiv.hide();
        warningsSpan.text("");
    }
    let errorsDiv = $("#errors" + index);
    let errorsSpan = $("#ELErrors" + index);
    if (currentElement.errors != null) {
        errorsDiv.show();
        errorsSpan.text(currentElement.errors.join("<br />"));
    } else {
        errorsDiv.hide();
        errorsSpan.text("");
    }
}

function setRate(rate, elName) {
    url = "/setElectrolyser/Production/" + elName + "/" + rate;
    $.ajax({
        method : "PUT",
        url: url
    });
}

function setOnOffButton(id, on) {
    btn = $("#"+id);
    btn.removeClass("ButtonChanging");
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
    Control.addClass("ButtonChanging")
    let putString = "/setElectrolyser/" + (Control.hasClass("swOff") ? "PowerOn" : "PowerOff") + "/" + id;
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
    }).fail(function (xhr, ajaxOptions, thrownError) {
        if (xhr.status === 400) {
            responseObj = JSON.parse(xhr.responseText)
            alert(responseObj.errors[0].Err);
        } else {
            alert(xhr.status + " : " + thrownError);
        }
    });
}

function RunClick(id, controlID, elName) {
    let Control = $("#"+controlID);
    Control.addClass("ButtonChanging")
    let putString = "/setElectrolyser/" + ((Control.hasClass("swOff")) ? "Start" : "Stop") + "/" + elName;
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function() {
            console.log("Electrolyser command sent OK");
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

function RebootClick(id, controlID, elName) {
    let Control = $("#"+controlID);
    Control.addClass("ButtonChanging")
    let putString = "/setElectrolyser/Reboot/" + elName;
    $.ajax({
        url: putString,
        type: 'put',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function() {
            console.log("Electrolyser reboot sent OK");
            alert(elName + " reboot command sent.");
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

function openElectrolyser(name, id) {

    if ($("#"+id).attr("on") === "1") {
        openElectrolyserMain(name);
    } else {
        window.open("ElectrolyserData.html?name=" + name);
    }
}

function openElectrolyserMain(name) {
    window.open("Electrolyser.html?name=" + name);
}
