function SetupPage() {
    $('#notes').jqxTextArea({
        placeHolder: 'Enter a note',
        height: 300,
        width: 500,
        minLength: 1,
        source: ""
    });
    $("#divStackSerial").hide();
}

function changeAction() {
    let serial = $("#divStackSerial");
    switch ($("#action").val()) {
        case "ReplaceStack" : serial.show();
        break;
        case "ReplaceSystem" : serial.hide();
        break;
        default : serial.hide();
    }
}

function saveSettings() {
    if (($("#action").val() === "ReplaceStack") && ($("#StackSerial").val() === "")) {
        alert("Please provide the new serial number of the new stack.");
        return;
    }
    $("#maintenanceForm").submit();
}