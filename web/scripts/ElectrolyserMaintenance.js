function SetupPage() {
    $('#notes').jqxTextArea({
        placeHolder: 'Enter a Country',
        height: 100,
        width: 200,
        minLength: 1,
        source: countries
    });
}