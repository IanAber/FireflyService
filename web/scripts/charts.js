var start = new Date();
start.setSeconds(0);
start.setMinutes(0);
start.setHours(0);
var end = new Date(start);
end.setHours(23);
end.setMinutes(59);
end.setSeconds(59);
var range = {from:start, to:end};

function buildURL() {
    end = new Date();
    end.setSeconds(0, 0);
    end = new Date(end.getTime() + 60000); // add 1 minute
    start = new Date(end - document.getElementById("timeRange").value);
    return buildURLForTimes(start, end);
}

function buildURLCustomTimes() {
    end = $("#endAt").jqxDateTimeInput('value');
    start = $("#startAt").jqxDateTimeInput('value');
    return buildURLForTimes(start, end);
}

function buildURLFocusTimes() {
    start = range.from;
    end = range.to;
    return buildURLForTimes(start, end);
}

function buildURLDoubleFocusTimes() {
    start = new Date();
    end = new Date();
    let from = range.from.getTime();
    let to = range.to.getTime();
    start.setTime(from - (to - from));
    end.setTime(to + (to - from));
    return buildURLForTimes(start, end);
}

function xAxisFormatFunction(value, _itemIndex, _series, _group) {
//    return value.toLocaleString("en-US", { hour12: true });
    return value.toLocaleTimeString();
}

function xAxisFormatDateFunction(value, _itemIndex, _series, _group) {
    return value.toLocaleDateString();
}

function xAxisFormatDateTimeFunction(value, _itemIndex, _series, _group) {
    return value.toLocaleString();
}

function xAxisSelectorFormatFunction(value, _itemIndex, _series, _group) {
    return value.toLocaleString();
}

function setupChart(Settings) {
    // select the chartContainer DIV element and render the chart.
    let chart = $('#ChartContainer');
    chart.jqxChart(Settings);
    chart.on('rangeSelectionChanged', function(evt) {
        range.from = evt.args.minValue;
        range.to = evt.args.maxValue;
    })

    sa = $("#startAt");
    ea = $("#endAt")
    sa.jqxDateTimeInput({ theme: "arctic", formatString: "F", showTimeButton: true, width: '300px', height: '25px' });
    sa.jqxDateTimeInput({ dropDownVerticalAlignment: 'top'});
    sa.css("float", "left");
    ea.jqxDateTimeInput({ theme: "arctic", formatString: "F", showTimeButton: true, width: '300px', height: '25px' });
    ea.jqxDateTimeInput({ dropDownVerticalAlignment: 'top'});
    ea.css("float", "left");
    getCurrent();
}

function refresh(url) {
    fetch(url)
        .then( function(response) {
            if (response.status === 200) {
                response.json()
                    .then(function(data) {
                        data.forEach(function (part, index) {
                            this[index].logged = new Date(part.logged * 1000);
                        }, data);

                        let Chart = $('#ChartContainer');
                        let waiting = $("#waiting");
                        if (data.length === 0) {
                            alert("No data found for the date/times selected");
                            waiting.hide();
                            return;
                        }
                        let start = data[0].logged;
                        let end = data[data.length - 1].logged;

                        end.setSeconds(0);
                        start.setSeconds(0);

                        let xAxis = Chart.jqxChart('xAxis');
                        xAxis.minValue = start;
                        xAxis.maxValue = end;
                        let scale = Math.round((end - start) / 600000) * 10;  // Scale in minutes
                        if (scale >= 5760) {
                            xAxis.unitInterval = Math.round(scale / 600);
                            xAxis.baseUnit = 'hour';
                            xAxis.formatFunction = xAxisFormatDateFunction;
                        } else if (scale >= 1440) {
                            xAxis.unitInterval = Math.round(scale / 600);
                            xAxis.baseUnit = 'hour';
                            xAxis.formatFunction = xAxisFormatDateTimeFunction;
                        } else {
//                            xAxis.unitInterval = Math.round(scale / 600);
                            xAxis.unitInterval = Math.round(scale / 10);
//                            xAxis.baseUnit = 'hour';
                            xAxis.baseUnit = 'minute';
                            xAxis.formatFunction = xAxisFormatFunction;
                        }
                        Chart.jqxChart('getInstance')._selectorRange = [];
                        Chart.jqxChart('update');
                        Chart.jqxChart({'source':data});
                        waiting.hide();
                        if (typeof postUpdate === "function") {
                            postUpdate(data);
                        }
                    });
            }
        })
        .catch(function(err) {
            if(err.name === "TypeError" && err.message !== "cancelled") {
                alert('Charging Fetch Error :-S' + err.message);
            }
        });
}

function goBack() {
    window.clearInterval(ChargingTimeout);
    if (window.history.length > 1) {
        setTimeout(window.history.back, 1000);
    } else {
        setTimeout(window.close, 1000);
    }
}

function getCurrent() {
    let TimeRange = $("#timeRange");
    let tr = parseInt(TimeRange.val());
    if (tr === 0)  {
        $("#customDateTimes").show();
        $("#waiting").show();
        refresh(buildURLCustomTimes());
    } else if (tr === 1) {
        $("#customDateTimes").show();
        $("#waiting").show();
        refresh(buildURLFocusTimes());
        TimeRange.val(0);
    } else if (tr === 2) {
        $("#customDateTimes").show();
        $("#waiting").show();
        refresh(buildURLDoubleFocusTimes());
        TimeRange.val(0);
    } else {
        $("#customDateTimes").hide();
        $("#waiting").show();
        refresh(buildURL());
    }
}

var RefreshTimer
function clickAutoRefresh(checkbox) {
    if (checkbox.checked) {
        if (RefreshTimer != null) {
            clearInterval(RefreshTimer)
        }
        RefreshTimer = setInterval(function() { getCurrent(); }, 5000);
    } else {
        if (RefreshTimer != null) {
            clearInterval(RefreshTimer)
            RefreshTimer = null
        }
    }
}