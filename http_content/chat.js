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

let connected = false;

function newEventSourceHandler() {
    if(typeof(EventSource) !== "undefined") {
        if (source != null) {
            console.log("closing event source gracefully");
            source.close();
            source = null;
            connected = false;
        }

        if (connected === true) {
            console.log("already connected. exiting.");
            return;
        }

        console.log("connecting to event source at /sse")
        source = new EventSource("/sse");
        connected = true;
        console.log("CONNECTED TO SSE");

        source.onerror = function(event) {
            console.error("UNEXPECTED ERROR. RECONNECTING source in 1s");
            if (source) { source.close(); }
            source = null;
            connected = false;
            console.error("UNEXPECTED ERROR. RECONNECTING source in 1s");
            setTimeout(newEventSourceHandler, 1000);
        };

        source.addEventListener('notify', notifyHandler, false);
        console.info("notifyHandler listener added");

        // schedule a reconnect in 30s
        console.info("Reconnecting in 30s...");
        setTimeout(newEventSourceHandler, 30000);
    } else {
        document.getElementById("result").innerHTML = "Sorry, your browser does not support server-sent events...";
    }
}


let source = null;
newEventSourceHandler();