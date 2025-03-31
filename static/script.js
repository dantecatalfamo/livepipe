const validateFilterPath = "/api/validate-filter"
const listChannelsPath = "/api/channels"
const createChannelPath = "/api/channels"
const getChannelHistoryPath = channel => `/api/channels/${channel}/history`
const updateChannelPath = channel => `/api/channels/${channel}`
const getChannelLivePath = channel => `/api/channels/${channel}/live`

async function listChannels() {
    const resp = await fetch(listChannelsPath)
    return await resp.json()
}

async function createChannel(name, filter) {
    const formData = new FormData()
    formData.append("name", name)
    formData.append("filter", filter)
    const resp = await fetch(createChannelPath, {
        method: "POST",
        body: formData
    })
    return await resp.json()
}

async function getChannelHistory(channel) {
    const resp = await fetch(getChannelHistoryPath(channel))
    return await resp.json()
}

async function setChannelFilter(channel, filter) {
    const formData = new FormData()
    formData.append("filter", filter)
    const resp = await fetch(updateChannelPath(channel), {
        method: "PATCH",
        body: formData,
    })
    if (!resp.ok) {
        const body = await resp.body.getReader().read()
        return String.fromCharCode.apply(null, body.value)
    }
}

async function setChannelName(channel, name) {
    const formData = new FormData()
    formData.append("name", name)
    const resp = await fetch(updateChannelPath(channel), {
        method: "PATCH",
        body: formData,
    })
    return await resp.json()
}


async function main() {
    const newChannelEl = document.getElementById("new-channel-form")
    const newChannelNameEl = document.getElementById("new-channel-name")
    const newChannelFilterEl = document.getElementById("new-channel-filter")
    newChannelEl.addEventListener("submit", async event => {
        event.preventDefault()
        const filter = newChannelFilterEl.value
        const name = newChannelNameEl.value
        const newChannel = await createChannel(name, filter)
        newChannelNameEl.value = ""
        newChannelFilterEl.value = ""
        channelsEl.appendChild(await channelBox(newChannel))
    })
    const channelsEl = document.getElementById("channels")
    const channels = await listChannels()
    for (channel of channels.channels) {
        channelsEl.appendChild(await channelBox(channel))
    }
}

async function channelBox(channel) {
    const channelEl = document.createElement("div")
    channelEl.classList.add("channel")

    const detailsEl = document.createElement("details")
    detailsEl.open = true
    channelEl.appendChild(detailsEl)

    const summaryEl = document.createElement("summary")
    detailsEl.appendChild(summaryEl)

    const nameEl = document.createElement("span")
    nameEl.classList.add("name")
    nameEl.innerText = channel.name
    summaryEl.appendChild(nameEl)

    if (channel.id != "stdin") {
        const filterEl = document.createElement("input")
        summaryEl.appendChild(filterEl)
        const filterErrorEl = document.createElement("span")
        filterErrorEl.classList.add("filter-error")
        summaryEl.appendChild(filterErrorEl)

        filterEl.classList.add("filter")
        filterEl.value = channel.filter
        filterEl.placeholder = "filter"
        filterEl.addEventListener("keydown", async event => {
            if (event.key === "Enter") {
                filterErrorEl.innerText = await setChannelFilter(channel.id, filterEl.value) || ""
            }
        })
    }

    const linesEl = document.createElement("div")
    linesEl.classList.add("lines")
    detailsEl.appendChild(linesEl)

    detailsEl.addEventListener("toggle", () => {
        linesEl.scrollTop = linesEl.scrollHeight
    })

    const anchorEl = document.createElement("div")
    anchorEl.classList.add("anchor")
    linesEl.appendChild(anchorEl)

    const channelHistory = await getChannelHistory(channel.id)

    for (line of channelHistory) {
        const lineEl = elementFromLine(line)
        linesEl.insertBefore(lineEl, anchorEl)
    }

    linesEl.scrollTop = linesEl.scrollHeight
    // https://css-tricks.com/books/greatest-css-tricks/pin-scrolling-to-bottom/
    linesEl.scroll(0, 1);

    const socket = new WebSocket(getChannelLivePath(channel.id))

    socket.addEventListener("message", event => {
        const json = JSON.parse(event.data)
        const lineEl = elementFromLine(json)
        linesEl.insertBefore(lineEl, anchorEl)
    })

    return channelEl
}

function elementFromLine(line) {
    const lineEl = document.createElement("div")
    const textEl = document.createElement("span")
    lineEl.classList.add("line")
    if (line.event) {
        lineEl.classList.add("event")
        textEl.innerText = line.event
    } else {
        textEl.innerText = line.text || ""
    }
    textEl.title = line.time
    lineEl.appendChild(textEl)

    return lineEl
}

main()
