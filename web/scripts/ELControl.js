// noinspection CommaExpressionJS
const ELDIV = `<div class="centered">
    <div class="control" id="el{{id}}" ondblclick="openElectrolyser('{{name}}', 'el{{id}}')">
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
                <img id="ELRun{{id}}" class="swOff" src="images/power-off.png" alt="Enable" onclick='RunClick({{id}}, "ELRun{{id}}", "{{name}}")' />
                <label for="ELRun{{id}}">Run</label></td>
            </div>
        </div>
    </div>
</div>`

var lockSliders = false;

function addElectrolyser(id, name, relay) {

    sCode = ELDIV.replace(/{{id}}/g, id).replace(/{{name}}/g, name).replace(/{{relay}}/g, relay);
    jQuery('#systems').append(sCode);
}

function updateElectrolyser(currentElement, index) {
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
    if (currentElement.on && currentElement.state !== 0) {
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

function openElectrolyser(name, id) {

    if ($("#"+id).attr("on") === "1") {
        window.open("Electrolyser.html?name=" + name);
    } else {
        window.open("ElectrolyserData.html?name=" + name);
    }
}
