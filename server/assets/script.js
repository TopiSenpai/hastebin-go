document.addEventListener("DOMContentLoaded", async () => {
    const matches = window.matchMedia("(prefers-color-scheme: dark)").matches;
    updateFaviconStyle(matches);

    const state = JSON.parse(document.getElementById("state").textContent);

    const params = new URLSearchParams(window.location.search);
    if (params.has("token")) {
        setToken(state.key, params.get("token"));
    }

    updateButtons(state);
    setState(state);
});

window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (event) => {
    updateFaviconStyle(event.matches);
});

window.addEventListener("popstate", (event) => {
    updateFiles(event.state);
    updateCode(event.state);
    updateButtons(event.state);
});


/* File Events */

document.getElementById("files").addEventListener("change", (e) => {
    const state = getState();
    state.current_file = parseInt(e.target.value);

    updateCode(state);
    setState(state);
})

document.getElementById("files").addEventListener("dblclick", (e) => {
    if (e.target.tagName.toLowerCase() !== "span") {
        return;
    }
    const state = getState();
    if (state.mode !== "edit") {
        return;
    }
    e.target.contentEditable = true;
    e.target.focus();
});

document.getElementById("files").addEventListener("focusout", (e) => {
    if (e.target.tagName.toLowerCase() !== "span") {
        return;
    }
    const state = getState();
    if (state.mode !== "edit") {
        return;
    }
    e.target.contentEditable = false;

    if (!state.files[state.current_file].name) {
        const name = `untitled${state.current_file > 0 ? state.current_file : ""}`;
        state.files[state.current_file].name = name;
        e.target.innerText = name;
    }
    setState(state);
});

document.getElementById("files").addEventListener("keypress", (e) => {
    const state = getState();
    if (state.mode !== "edit") {
        return;
    }
    if (e.key === "Enter") {
        e.preventDefault();
        if (e.target.contentEditable) {
            e.target.blur();
        }
    }
});

document.getElementById("files").addEventListener("input", (e) => {
    if (e.target.name === "files") {
        return;
    }
    const state = getState();
    state.files[state.current_file].name = e.target.innerText.trim();
    setState(state);
})

document.getElementById("files").addEventListener("click", (e) => {
    if (e.target.tagName.toLowerCase() !== "button") {
        return;
    }
    const state = getState();
    const index = parseInt(document.getElementById(e.target.parentElement.htmlFor).value);
    state.files.splice(index, 1);

    if (index === state.current_file) {
        state.current_file = 0;
    }

    if (state.files.length === 0) {
        state.files.push({
            name: "untitled",
            content: "",
            formatted: "",
            language: "auto"
        })
    }

    updateFiles(state);
    updateCode(state);
    setState(state);
})

document.getElementById("file-add").addEventListener("click", (e) => {
    const state = getState();
    const index = state.files.length;

    state.files[index] = {
        name: `untitled${index}`,
        content: "",
        formatted: "",
        language: "auto"
    }

    updateFiles(state)
    setState(state);
    document.querySelector(`label[for="file-${index}"]`).click();
});


/* Code Edit Events */

document.getElementById("code-edit").addEventListener("keydown", (e) => {
    if (e.key !== "Tab" || e.shiftKey) {
        return;
    }
    e.preventDefault();

    const start = e.target.selectionStart;
    const end = e.target.selectionEnd;
    e.target.value = e.target.value.substring(0, start) + "\t" + e.target.value.substring(end);
    e.target.selectionStart = e.target.selectionEnd = start + 1;
});

document.getElementById("code-edit").addEventListener("input", (e) => {
    const state = getState();
    state.files[state.current_file].content = e.target.value;

    const count = state.files.reduce((total, file) => total + file.content.length, 0);
    document.getElementById("code-edit-count").innerHTML = `${count}`
    const maxElement = document.getElementById("code-edit-max");
    if (!maxElement) return;
    document.querySelector(`label[for="code-edit"]`).classList.toggle("invalid", count > maxElement.innerHTML.substring(1));
});

document.getElementById("code-edit").addEventListener("paste", (event) => {
    const state = getState();
    state.files[state.current_file].content = event.target.value;
    updateButtons(state);
    setState(state);
})

document.getElementById("code-edit").addEventListener("cut", (event) => {
    const state = getState();
    state.files[state.current_file].content = event.target.value;
    updateButtons(state);
    setState(state);
})

document.getElementById("code-edit").addEventListener("keyup", (event) => {
    const state = getState();
    state.files[state.current_file].content = event.target.value;
    updateButtons(state);
    setState(state);
})

/* Footer Events */

document.getElementById("version").addEventListener("change", async (e) => {
    const state = getState();

    let newVersion = e.target.value;
    if (newVersion === state.version) {
        return;
    }
    if (e.target.options.item(0).value === newVersion) {
        newVersion = 0;
    }

    const document = await fetchDocument(state.key, newVersion);
    if (!document) {
        return;
    }

    state.version = document.version;
    state.files = document.files;
    if (state.current_file >= state.files.length) {
        state.current_file = state.files.length - 1;
    }

    updateVersionSelect(e.target.selectedIndex);

    updateFiles(state)
    updateCode(state)

    addState(state)
});

document.getElementById("theme").addEventListener("change", async (e) => {
    const theme = e.target.value;
    const colorScheme = e.target.options.item(e.target.selectedIndex).dataset.colorScheme;

    setCookie("theme", theme);
    document.documentElement.setAttribute("data-theme", colorScheme);
    document.documentElement.classList.replace(colorScheme === "dark" ? "light" : "dark", colorScheme);

    const themeCssElement = document.getElementById("theme-css");
    const href = new URL(themeCssElement.href);
    href.searchParams.set("theme", theme);
    themeCssElement.href = href.toString();

    const state = getState();

    if (!state.key) {
        return;
    }
    const doc = await fetchDocument(state.key, state.version);
    if (!doc) {
        return;
    }

    state.files = doc.files;

    updateFiles(state)
    updateCode(state)

    addState(state)
});

document.getElementById("folds").addEventListener("change", async (e) => {
    const checked = e.target.checked;

    setCookie("folds", checked);

    const state = getState();

    const doc = await fetchDocument(state.key, state.version);
    if (!doc) {
        return;
    }

    state.files = doc.files;

    updateFiles(state)
    updateCode(state)

    addState(state)
})


document.getElementById("expire").addEventListener("input", (e) => {
    const expireIn = parseInt(e.target.value);
    const invalid = isNaN(expireIn);

    document.getElementById("expire").classList.toggle("invalid", invalid);
    if (invalid) {
        return;
    }
    const state = getState();
    state.expire_in = expireIn;
    setState(state);
});

document.getElementById("language").addEventListener("change", async (e) => {
    const state = getState();
    const file = state.files[state.current_file];
    file.language = e.target.value;

    if (state.mode === "view") {
        state.files[state.current_file] = await fetchDocumentFile(state.key, state.version, file.name, file.language);
        updateCode(state);
    }
    setState(state);
});

/* Keyboard Shortcut Events */

document.addEventListener("keydown", (event) => {
    const shortcuts = {s: "save", n: "new", e: "edit", d: "duplicate"};
    if (!event.ctrlKey || !(event.key in shortcuts)) return;
    doKeyboardAction(event, shortcuts[event.key]);
})

const doKeyboardAction = (event, elementId) => {
    event.preventDefault();
    if (document.getElementById(elementId).disabled) return;
    document.getElementById(elementId).click();
}

/* Navigation Action Button Events */

document.getElementById("edit").addEventListener("click", async () => {
    if (document.getElementById("edit").disabled) return;

    const state = getState();
    if (!hasPermission(getToken(state.key), PermissionWrite)) {
        state.key = "";
    }
    state.mode = "edit";
    state.version = 0;

    updateCode(state);
    updateButtons(state);
    addState(state)
});

document.getElementById("save").addEventListener("click", async () => {
    if (document.getElementById("save").disabled) {
        return;
    }
    const state = getState();
    if (state.mode !== "edit") {
        return;
    }

    const saveButton = document.getElementById("save");
    saveButton.classList.add("loading");
    const doc = await saveDocument(state.key, state.expire_in, state.files);
    saveButton.classList.remove("loading");

    if (!doc) {
        return;
    }
    state.key = doc.key;
    state.version = 0;
    state.files = doc.files;
    state.mode = "view";
    state.expire_in = 0;

    if (doc.token) {
        setToken(doc.key, doc.token);
    }

    const optionElement = document.createElement("option");
    optionElement.title = `${doc.version_time}`;
    optionElement.value = doc.version;
    optionElement.innerText = `${doc.version_label}`;

    updateVersionSelect(-1);
    const versionElement = document.getElementById("version");
    versionElement.insertBefore(optionElement, versionElement.firstChild);
    versionElement.value = doc.version;

    document.getElementById("expire").value = "";

    updateCode(state);
    updateButtons(state);
    addState(state);
});

document.getElementById("delete").addEventListener("click", async () => {
    if (document.getElementById("delete").disabled) {
        return;
    }

    const state = getState();
    const token = getToken(state.key);
    if (!token) {
        return;
    }

    const deleteConfirm = window.confirm("Are you sure you want to delete this document? This action cannot be undone.")
    if (!deleteConfirm) {
        return;
    }

    const deleteButton = document.getElementById("delete");
    deleteButton.classList.add("loading");
    await deleteDocument(state.key, token)
    deleteButton.classList.remove("loading");

    deleteToken(state.key);

    state.key = "";
    state.vesion = 0;
    state.mode = "edit"
    state.files = [{
        name: "untitled",
        content: "",
        formatted: "",
        language: "auto"
    }];
    state.file_selected = 0;

    updateCode(state);
    updateButtons(state);
    addState(state);
})

document.getElementById("copy").addEventListener("click", async () => {
    if (document.getElementById("copy").disabled) {
        return;
    }

    const state = getState();
    await navigator.clipboard.writeText(state.files[state.current_file].content);
})

document.getElementById("raw").addEventListener("click", () => {
    if (document.getElementById("raw").disabled) {
        return;
    }

    const {key, version} = getState();
    if (!key) return;
    window.open(`/raw/${key}${version !== 0 ? `/versions/${version}` : ""}`, "_blank").focus();
})

document.getElementById("share").addEventListener("click", async () => {
    if (document.getElementById("share").disabled) return;

    const {key} = getState();
    const token = getToken(key);
    if (!hasPermission(token, PermissionShare)) {
        await navigator.clipboard.writeText(window.location.href);
        return;
    }

    document.getElementById("share-permissions-write").checked = false;
    document.getElementById("share-permissions-delete").checked = false;
    document.getElementById("share-permissions-share").checked = false;

    document.getElementById("share-dialog").showModal();
});

document.getElementById("share-dialog-close").addEventListener("click", () => {
    document.getElementById("share-dialog").close();
});

document.getElementById("share-copy").addEventListener("click", async () => {
    const permissions = [];
    if (document.getElementById("share-permissions-write").checked) {
        permissions.push("write");
    }
    if (document.getElementById("share-permissions-delete").checked) {
        permissions.push("delete");
    }
    if (document.getElementById("share-permissions-share").checked) {
        permissions.push("share");
    }
    if (document.getElementById("share-permissions-webhook").checked) {
        permissions.push("webhook");
    }

    if (permissions.length === 0) {
        await navigator.clipboard.writeText(window.location.href);
        document.getElementById("share-dialog").close();
        return;
    }

    const {key} = getState();
    const token = getToken(key);

    const response = await fetch(`/documents/${key}/share`, {
        method: "POST",
        body: JSON.stringify({permissions: permissions}),
        headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`
        }
    });

    if (!response.ok) {
        const body = await response.json();
        showErrorPopup(body.message || response.statusText)
        console.error("error sharing document:", response);
        return;
    }

    const body = await response.json()
    const shareUrl = window.location.href + "?token=" + body.token;
    await navigator.clipboard.writeText(shareUrl);
    document.getElementById("share-dialog").close();
});

document.getElementById("documents").addEventListener("click", async () => {
    const tokens = getTokens();
    if (!tokens) {
        return;
    }
    console.log(tokens)

    const documents = document.getElementById("documents-list");
    documents.replaceChildren();

    for (const [key, token] of Object.entries(tokens)) {
        const tokenSplit = token.split(".")
        if (tokenSplit.length !== 3) {
            continue;
        }
        const data = JSON.parse(atob(tokenSplit[1]))
        const date = new Date(data.iat * 1000);
        const formattedDate = date.toLocaleString("en-US", { year: "numeric", month: "long", day: "numeric", hour: "2-digit", minute: "2-digit" });

        const li = document.createElement("li");
        li.innerHTML = `<span>${key} ${formattedDate}</span><button class="documents-open"></button><button class="documents-remove"></button>`;

        documents.appendChild(li);
    }

    document.getElementById("documents-dialog").showModal();
});

document.getElementById("documents-dialog-close").addEventListener("click", () => {
    document.getElementById("documents-dialog").close();
});

async function saveDocument(key, expire, files) {
    const data = new FormData();
    for (const [i, file] of files.entries()) {
        const blob = new Blob([file.content], {
            type: file.language,
        })
        data.append(`file-${i}`, blob, file.name);
    }

    const headers = {};
    const token = getToken(key);
    if (token) {
        headers["Authorization"] = `Bearer ${token}`
    }

    if (expire) {
        try {
            headers["Expires"] = new Date(Date.now() + expire * 60 * 60 * 1000).toISOString();
        } catch (e) {
            showErrorPopup("Invalid expiration date");
            return
        }
    }

    const response = await fetch(`/documents/${key}?formatter=html`, {
        body: data,
        method: key !== "" ? "PATCH" : "POST",
        headers: headers
    });

    let body = await response.text();
    try {
        body = JSON.parse(body);
    } catch (e) {
        body = {message: body};
    }

    if (!response.ok) {
        showErrorPopup(body.message || response.statusText);
        console.error("error saving document:", response);
        return;
    }

    return body
}

async function fetchDocument(key, version) {
    const response = await fetch(`/documents/${key}${version !== 0 ? `/versions/${version}` : ""}?formatter=html`, {
        method: "GET"
    });

    let body = await response.text();
    try {
        body = JSON.parse(body);
    } catch (e) {
        body = {message: body};
    }
    if (!response.ok) {
        showErrorPopup(body.message || response.statusText);
        console.error("error fetching document version:", response);
        return;
    }

    return body
}

async function fetchDocumentFile(key, version, file, language) {
    const response = await fetch(`/documents/${key}${version !== 0 ? `/versions/${version}` : ""}/files/${file}?formatter=html&language=${language}`, {
        method: "GET"
    });

    let body = await response.text();
    try {
        body = JSON.parse(body);
    } catch (e) {
        body = {message: body};
    }
    if (!response.ok) {
        showErrorPopup(body.message || response.statusText);
        console.error("error fetching document version:", response);
        return;
    }

    return body
}

async function deleteDocument(key, token) {
    const response = await fetch(`/documents/${key}`, {
        method: "DELETE",
        headers: {
            Authorization: `Bearer ${token}`
        }
    });

    if (response.status === 204) {
        return;
    }

    let body = await response.text();
    try {
        body = JSON.parse(body);
    } catch (e) {
        body = {message: body};
    }
    if (!response.ok) {
        showErrorPopup(body.message || response.statusText);
        console.error("error fetching document version:", response);
    }
}

function showErrorPopup(message) {
    const popup = document.getElementById("error-popup");
    popup.style.display = "block";
    popup.innerText = message || "Something went wrong.";
    setTimeout(() => popup.style.display = "none", 5000);
}

function getState() {
    return window.history.state;
}

function getURL(state) {
    const url = new URL(window.location.href);
    if (state.files.length > 1) {
        url.searchParams.set("file", state.files[state.current_file].name);
    } else {
        url.searchParams.delete("file");
    }
    url.pathname = `/${state.key}${state.version !== 0 ? `/${state.version}` : ""}`;
    return url.toString();
}

function setState(state) {
    window.history.replaceState(state, "", getURL(state))
}

function addState(state) {
    window.history.pushState(state, "", getURL(state))
}

function getToken(key) {
    const documents = localStorage.getItem("documents")
    if (!documents) return ""
    const token = JSON.parse(documents)[key]
    if (!token) return ""

    return token
}

function getTokens() {
    const documents = localStorage.getItem("documents")
    if (!documents) {
        return;
    }
    return JSON.parse(documents)
}

function setToken(key, token) {
    let documents = localStorage.getItem("documents")
    if (!documents) {
        documents = "{}"
    }
    const parsedDocuments = JSON.parse(documents)
    parsedDocuments[key] = token
    localStorage.setItem("documents", JSON.stringify(parsedDocuments))
}

function deleteToken(key) {
    const documents = localStorage.getItem("documents");
    if (!documents) return;
    const parsedDocuments = JSON.parse(documents);
    delete parsedDocuments[key]
    localStorage.setItem("documents", JSON.stringify(parsedDocuments));
}

const PermissionWrite = 1
const PermissionDelete = 2
const PermissionShare = 4
const PermissionWebhook = 8

function hasPermission(token, permission) {
    if (!token) return false;
    const tokenSplit = token.split(".")
    if (tokenSplit.length !== 3) return false;
    return (JSON.parse(atob(tokenSplit[1])).pms & permission) === permission;
}

function updateVersionSelect(currentIndex) {
    const versionElement = document.getElementById("version")
    for (let i = 0; i < versionElement.options.length; i++) {
        const element = versionElement.options.item(i);
        if (element.innerText.endsWith(" (current)")) {
            element.innerText = element.innerText.substring(0, element.innerText.length - 10);
        }
    }
    if (currentIndex !== versionElement.options.length - 1 && currentIndex !== -1) {
        versionElement.options.item(currentIndex).innerText += " (current)";
    }
}

function updateFiles(state) {
    const nodes = [];
    for (const [i, file] of state.files.entries()) {
        const input = document.createElement("input");
        input.id = `file-${i}`;
        input.type = "radio";
        input.name = "files";
        input.value = `${i}`;
        if (i === state.current_file) {
            input.checked = true;
        }

        const label = document.createElement("label");
        label.htmlFor = `file-${i}`;
        label.innerHTML += `<span>${file.name}</span><button class="file-remove" ${state.mode === "view" ? "disabled" : ""}></button>`;

        nodes.push(input);
        nodes.push(label);
    }

    const files = document.getElementById("files");
    nodes.push(files.lastElementChild);

    files.replaceChildren(...nodes);
}

function updateCode(state) {
    if (!state) return;

    const codeElement = document.getElementById("code");
    const codeEditElement = document.getElementById("code-edit");

    if (state.mode === "view") {
        codeEditElement.style.display = "none";
        codeElement.style.display = "block";
    } else {
        codeEditElement.style.display = "block";
        codeElement.style.display = "none";
    }

    const file = state.files[state.current_file];
    document.getElementById("code-edit").value = file.content;
    document.getElementById("code-view").innerHTML = file.formatted;
    document.getElementById("language").value = file.language;
}

function updateButtons(state) {
    const token = getToken(state.key);
    // update page title
    if (state.key) {
        document.title = `gobin - ${state.key}`;
    } else {
        document.title = "gobin";
    }

    document.querySelectorAll(".file-remove").forEach((element) => element.disabled = state.mode === "view");

    const fileAddButton = document.getElementById("file-add");
    const saveButton = document.getElementById("save");
    const editButton = document.getElementById("edit");
    const deleteButton = document.getElementById("delete");
    const copyButton = document.getElementById("copy");
    const rawButton = document.getElementById("raw");
    const shareButton = document.getElementById("share");
    const expireLabel = document.querySelector(`label[for="expire"]`);
    const versionSelect = document.getElementById("version");
    versionSelect.disabled = versionSelect.options.length <= 1;
    if (state.mode === "view") {
        fileAddButton.style.display = "none";
        saveButton.style.display = "none";
        editButton.style.display = "block";
        deleteButton.disabled = !hasPermission(token, PermissionDelete);
        copyButton.disabled = false;
        rawButton.disabled = false;
        shareButton.disabled = false;
        expireLabel.style.display = "none";
        return;
    }
    fileAddButton.style.display = "block";
    saveButton.style.display = "block";
    saveButton.disabled = state.files.findIndex(file => file.content.length > 0) === -1;
    editButton.style.display = "none";
    deleteButton.disabled = true;
    copyButton.disabled = true;
    rawButton.disabled = true;
    shareButton.disabled = true;
    expireLabel.style.display = "block";
}

function updateFaviconStyle(matches) {
    const faviconElement = document.querySelector(`link[rel="icon"]`)
    if (matches) {
        faviconElement.href = "/assets/favicon.png";
        return
    }
    faviconElement.href = "/assets/favicon-light.png";
}

function setCookie(name, value, options = {}) {
    options = {
        path: "/",
        sameSite: "strict",
        ...options
    };

    if (options.expires instanceof Date) {
        options.expires = options.expires.toUTCString();
    }

    let updatedCookie = encodeURIComponent(name) + "=" + encodeURIComponent(value);

    for (let optionKey in options) {
        updatedCookie += "; " + optionKey;
        let optionValue = options[optionKey];
        if (optionValue !== true) {
            updatedCookie += "=" + optionValue;
        }
    }

    document.cookie = updatedCookie;
}
