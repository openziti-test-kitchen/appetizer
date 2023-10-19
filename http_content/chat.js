function fadeOutElement(element, duration) {
    var opacity = 1;
    var interval = 5;
    var step = (interval / duration);

    var fadeOut = setInterval(function() {
        if (opacity <= 0.05) {
            clearInterval(fadeOut);
            element.style.display = 'none';
        } else {
            opacity -= step;
            element.style.opacity = opacity;
        }
    }, interval);
}

function notifyHandler(event) {
    const parts = event.data.split(':');
    const who = parts[0];
    const what = parts.slice(1).join(':');

    let d = document.createElement("div");
    d.className = "chat";
    d.innerHTML = "<p class=\"chatter\">" + who + "</p><p class=\"comment\">" + what + "</p>";

    let bubs = document.getElementById("bubs");
    bubs.append(d);
    setTimeout(fadeOutElement, 30000, d, 1000);
}

function newEventSourceHandler() {
    if(typeof(EventSource) !== "undefined") {
        if (source) {
            console.log("closing event source gracefully")
            source.close();
        }
        console.log("closing event source gracefully")
        source = new EventSource("/sse");
        console.log("CONNECTED TO SSE")

        source.onerror = function(event) {
            console.error("UNEXPECTED ERROR. RECONNECTING source in 1s")
            if (source) { source.close(); }
            console.error("UNEXPECTED ERROR. RECONNECTING source in 1s")
            setTimeout(newEventSourceHandler, 1000);
        };

        source.addEventListener('notify', notifyHandler, false);
        console.info("notifyHandler listener added")
    } else {
        document.getElementById("result").innerHTML = "Sorry, your browser does not support server-sent events...";
    }
    setTimeout(newEventSourceHandler, 30)
}


let source;
newEventSourceHandler()