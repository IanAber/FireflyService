function showFuelCellAlarms(data, alarmDiv) {
    let alarmText;
    if (data.Alarms.length > 0) {
        alarmText = '<span class="alarm">';
        alarmText += data.Alarms.join('</span><br /><span class="alarm">')
        alarmText += '</span>'
    } else {
        alarmText = "";
    }
    if (data.DCOutputStatus === "fault") {
        alarmText += '<span class="alarm">' + data.DCOutputFault + '</span><br />'
    }
    if (alarmText !== "") {
        alarmDiv.html(alarmText);
        alarmDiv.show();
    } else {
        alarmDiv.hide();
    }
}
