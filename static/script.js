const MAX_LINES = 1_000

const validateFilterPath = "/api/validate-filter"
const listChannelsPath = "/api/channels"
const createChannelPath = "/api/channels"
const getChannelHistoryPath = channel => `/api/channels/${channel}/history`
const getChannelPlainPath = channel => `/api/channels/${channel}/plain`
const updateChannelPath = channel => `/api/channels/${channel}`
const deleteChannelPath = channel => `/api/channels/${channel}`
const getChannelLivePath = channel => `/api/channels/${channel}/live`

async function listChannels() {
    const resp = await fetch(listChannelsPath)
    return await resp.json()
}

async function createChannel(name, filter, replace) {
    const formData = new FormData()
    formData.append("name", name)
    formData.append("filter", filter)
    formData.append("replace", replace)
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

async function setChannelReplace(channel, replace) {
    const formData = new FormData()
    formData.append("replace", replace)
    await fetch(updateChannelPath(channel), {
        method: "PATCH",
        body: formData,
    })
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

async function deleteChannel(channel) {
    const resp = await fetch(deleteChannelPath(channel), {
        method: "DELETE"
    })
    return resp.ok
}


async function main() {
    const newChannelEl = document.getElementById("new-channel-form")
    const newChannelNameEl = document.getElementById("new-channel-name")
    const newChannelFilterEl = document.getElementById("new-channel-filter")
    const newChannelReplaceEl = document.getElementById("new-channel-replace")
    newChannelEl.addEventListener("submit", async event => {
        event.preventDefault()
        const filter = newChannelFilterEl.value
        const name = newChannelNameEl.value
        const replace = newChannelReplaceEl.value
        const newChannel = await createChannel(name, filter, replace)
        newChannelNameEl.value = ""
        newChannelFilterEl.value = ""
        newChannelReplaceEl.value = ""
        channelsEl.appendChild(await channelBox(newChannel))
    })
    const channelsEl = document.getElementById("channels")
    const channels = await listChannels()
    for (channel of channels.channels) {
        channelsEl.appendChild(await channelBox(channel))
    }
}

async function channelBox(channel) {
    const channelEl = document.getElementById("channel-template").content.cloneNode(true).querySelector(".channel")
    const detailsEl = channelEl.querySelector("details")
    const nameEl = channelEl.querySelector(".name")
    const filterEl = channelEl.querySelector(".filter")
    const filterErrorEl = channelEl.querySelector(".error")
    const replaceEl = channelEl.querySelector(".replace")
    const downloadEl = channelEl.querySelector(".download")
    const deleteEl = channelEl.querySelector(".delete")
    const linesEl = channelEl.querySelector(".lines")
    const anchorEl = channelEl.querySelector(".anchor")

    nameEl.innerText = channel.name

    filterEl.value = channel.filter
    filterEl.addEventListener("keydown", async event => {
        if (event.key === "Enter") {
            filterErrorEl.innerText = await setChannelFilter(channel.id, filterEl.value) || ""
        }
    })

    replaceEl.value = channel.replace
    replaceEl.addEventListener("keydown", async event => {
        if (event.key == "Enter") {
            await setChannelReplace(channel.id, replaceEl.value)
        }
    })

    downloadEl.href = getChannelPlainPath(channel.id)

    deleteEl.addEventListener("click", async () => {
        if (await deleteChannel(channel.id)) {
            channelEl.remove()
        }
    })

    if (channel.id == "stdin") {
        filterEl.remove()
        replaceEl.remove()
    }

    if (channel.id == "stdin" || channel.id == "stdout") {
        deleteEl.remove()
    }

    detailsEl.addEventListener("toggle", () => {
        linesEl.scrollTop = linesEl.scrollHeight
    })

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
        if (linesEl.childElementCount > MAX_LINES) {
            const removeCount = linesEl.childElementCount-MAX_LINES;
            while (removeCount > 0) {
                linesEl.firstChild.remove()
                removeCount--
            }
        }
        linesEl.insertBefore(lineEl, anchorEl)
    })

    socket.addEventListener("error", event => {
        console.error(`socket ${channel.name} error`, event)
        linesEl.insertBefore(elementFromLine({event: "socket error"}), anchorEl)
        socket.close("socket error")
    })

    socket.addEventListener("close", event => {
        console.error(`socket ${channel.name} close`, event.reason)
        nameEl.classList.add("error")
        linesEl.insertBefore(elementFromLine({event: "socket closed"}), anchorEl)
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
    textEl.title = new Date(line.time)
    lineEl.appendChild(textEl)

    return lineEl
}

main()
