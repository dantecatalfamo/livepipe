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
    const resp = await fetch(createChannelPath, {
        method: "POST",
        body: JSON.stringify({ name, filter })
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
    return await resp.json()
}

async function setChannelFilter(channel, name) {
    const formData = new FormData()
    formData.append("name", name)
    const resp = await fetch(updateChannelPath(channel), {
        method: "PATCH",
        body: formData,
    })
    return await resp.json()
}


async function main() {
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

    const filterEl = document.createElement("input")
    filterEl.classList.add("filter")
    filterEl.value = channel.filter
    summaryEl.appendChild(filterEl)

    const linesEl = document.createElement("div")
    linesEl.classList.add("lines")
    detailsEl.appendChild(linesEl)

    const anchorEl = document.createElement("div")
    anchorEl.classList.add("anchor")
    linesEl.appendChild(anchorEl)

    const channelHistory = await getChannelHistory(channel.id)

    for (line of channelHistory) {
        const lineEl = elementFromLine(line)
        linesEl.insertBefore(lineEl, anchorEl)
    }

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
    textEl.innerText = line.text
    textEl.title = line.time
    lineEl.appendChild(textEl)

    return lineEl
}

main()
