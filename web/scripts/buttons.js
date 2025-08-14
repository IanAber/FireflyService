function RenderButtons(buttons) {
    const controls = $("#buttonsDiv");
    let buttonCount = 0;
    for (let button of buttons) {
        if (button.ShowOnCustomer) {
            buttonCount++
        }
    }
    if (controls.children().length === buttonCount) {
        let buttonId = 0;
        for (let button of buttons) {
            if (button.ShowOnCustomer) {
                const btn = $("#buttonDiv" + buttonId);

                if (button.Pressed) {
                    btn.removeClass("ButtonChanging");
                    btn.removeClass("ButtonOff");
                    btn.addClass("ButtonOn");
                } else {
                    btn.removeClass("ButtonChanging");
                    btn.removeClass("ButtonOn");
                    btn.addClass("ButtonOff");
                }
            }
            buttonId++;
        }
    } else {
        controls.children().remove();
        let buttonId = 0;
        let buttonTag;
        for (let button of buttons) {
            if (button.ShowOnCustomer) {
                let buttonClass = button.Pressed ? 'ButtonOn' : 'ButtonOff'
                buttonTag = `<div class="button ${buttonClass}" onclick="clickButton(${buttonId})" id="buttonDiv${buttonId}"><span class="button" id="button${buttonId}">${button.Name}</span></div>`;
                controls.append(buttonTag);
                const btn = document.getElementById("buttonDiv" + buttonId);
                btn.addEventListener("contextmenu", (e) => {e.preventDefault()});
            }
            buttonId++;
        }
    }
}
