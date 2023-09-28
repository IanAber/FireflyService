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
    end = new Date(end.getTime() + 60000);
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

function xAxisSelectorFormatFunction(value, itemIndex, series, group) {
    return value.toLocaleString();
}

function setupChart(Settings) {
    // select the chartContainer DIV element and render the chart.
    $('#ChartContainer').jqxChart(Settings);
    $('#ChartContainer').on('rangeSelectionChanged', function(evt) {
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
                        end = Math.trunc($("#endAt").jqxDateTimeInput('value')); // / 1000);
                        start = ($("#startAt").jqxDateTimeInput('value') ); /// 1000)
                        interval = Math.round((end - start) / 30);
                        range.min = start;
                        range.max = end;

                        let Chart = $('#ChartContainer');
                        let xAxis = Chart.jqxChart('xAxis');
                        xAxis.minValue = start;
                        xAxis.maxValue = end;
                        Chart.jqxChart({'source':data});
                        Chart.jqxChart('getInstance')._selectorRange = [];
                        Chart.jqxChart('update');
                        $("#waiting").hide();
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
    let tr = parseInt($("#timeRange").val());
    if (tr === 0)  {
        $("#customDateTimes").show();
        $("#waiting").show();
        refresh(buildURLCustomTimes());
    } else if (tr === 1) {
        $("#customDateTimes").show();
        $("#waiting").show();
        refresh(buildURLFocusTimes());
        $("#timeRange").val(0);
    } else if (tr === 2) {
        $("#customDateTimes").show();
        $("#waiting").show();
        refresh(buildURLDoubleFocusTimes());
        $("#timeRange").val(0);
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