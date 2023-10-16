
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
    setTimeout(fadeOutElement, 10000, d, 1000);
}

if(typeof(EventSource) !== "undefined") {
    let source = new EventSource("https://appetizer.openziti.io/sse");
    console.log("CONNECTED TO SSE")

    source.onerror = function(event) {
        setTimeout(function() {
            source = new EventSource('/sse'); // Re-establish the connection
        }, 2000); // Adjust the delay as needed
        console.log("ERROR RECONNECTING")
    };

    source.addEventListener('notify', notifyHandler, false);
} else {
    document.getElementById("result").innerHTML = "Sorry, your browser does not support server-sent events...";
}