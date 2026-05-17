const editable = document.querySelector('.message-input');
const placeholder = document.querySelector('.placeholder');
const connectBtn = document.getElementById("connectButton")
const id = document.getElementById("nickname");
const nickname = document.getElementById("nickname");
const input = document.getElementById("messageInput");
const channels = document.getElementById("channels");
const activeRoom = document.getElementById("activeRoom");
const chat = document.getElementById("chatbox");
const msgTemplate = document.getElementById("message-template");
const chnTemplate = document.getElementById("channel-template");

const EventChatMessage = "chat.message";
const EventUserStatus  = "user.status";
const EventJoinRoom    = "user.joinRoom";
const EventFetchRooms  = "user.fetchRooms";
const EventRoomsList   = "user.roomsList";

let socket = null;

const state = {
    currentRoom: null,
    rooms: new Map(),
}

function updatePlaceholder() {
    placeholder.style.display =
        editable.textContent.length === 0 ? 'block' : 'none';
}

function renderMessage(author, content) {
    const clone = msgTemplate.content.cloneNode(true);
    const shouldScroll = isUserAtBottom(chat);

    clone.querySelector(".message-header").textContent = author;
    clone.querySelector(".message-body").textContent = content;

    chat.appendChild(clone);

    if (shouldScroll) {
        chat.scrollTop = chat.scrollHeight;
    }
}

function renderRoom(roomId) {
    const clone = chnTemplate.content.cloneNode(true);
    const button = clone.firstElementChild;

    button.dataset.roomId = roomId;
    clone.querySelector(".chn-header").textContent = "Комната";
    clone.querySelector(".sub-text").textContent = roomId;

    button.addEventListener("click", () => {
        switchRoom(roomId);
    });

    // room.element = button;
    // state.rooms.set(room.id, room);

    channels.appendChild(button);
    return button;
}

function addRoom(roomId) {
    if (state.rooms.has(roomId)) return state.rooms.get(roomId);

    const room = {
        id: roomId,
        messages: [],
        element: renderRoom(roomId)
    };

    state.rooms.set(roomId, room);
    // state.rooms.set(roomData.id, {id: roomData.id, element: null, messages: []});

    return room;
}

function addMessage(roomId, author, content) {
    const room = addRoom(roomId);
    room.messages.push({author, content})

    if (room.element) {
        channels.prepend(room.element);
    }

    if (state.currentRoom == roomId) {
        renderMessage(author, content)
    }
}

function switchRoom(roomId) {
    state.currentRoom = roomId;
    activeRoom.value = roomId;

    chat.innerHTML = "";
    const room = state.rooms.get(roomId)
    if (room) {
        for (const msg of room.messages) {
            renderMessage(msg.author, msg.content)
        };
    };

    // визуальное выделение
    document.querySelectorAll(".chn-btn").forEach(btn => {
        btn.classList.toggle("active", btn.dataset.roomId === roomId);
    });
}

function isUserAtBottom(element, threshold = 5) {
    return element.scrollHeight - element.scrollTop - element.clientHeight < threshold;
}

function connect() {
    if (!socket || socket.readyState === WebSocket.CLOSED) {
        socket = new WebSocket(`ws://192.168.0.8:8080/ws?id=${id.value}`);

        socket.onopen = function () {
            connectBtn.value = "Отключиться"
        };

        socket.onmessage = function (e) {
            try {
                const event = JSON.parse(e.data);
                switch (event.type) {
                    case EventChatMessage:
                        addMessage(event.data.room, event.data.author, event.data.content);
                        break;

                    case EventUserStatus:
                        renderMessage(event.data.user, "Теперь " + event.data.state);
                        break;
                    case EventJoinRoom:
                        renderMessage(event.data.user, "Присоединился к комнате " + event.data.room);
                        break;
                    case EventRoomsList:
                        event.data.rooms.forEach(roomData => {
                            addRoom(roomData.id)
                        });
                        break
                    default:
                        console.log("Неизвестный тип сообщения: " + event.type);
                        break;
                }
            } catch (err) {
                console.log(err);
            }
        };

        socket.onclose = () => {
            console.log("Соединение закрыто");
            socket = null; // Очищаем ссылку
            connectBtn.value = "Подключиться";
        };

    } else {
        socket.close()
    }
}

function sendMessage() {
    const msg = {
        type: "chat.message",
        data: {
            content: input.textContent,
            room: activeRoom.value,
        },
    };

    send(JSON.stringify(msg));

    input.textContent = "";
    updatePlaceholder();
}

function send(data) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(data);
    }
}

function joinRoom() {
    const msg = {
        type: EventJoinRoom,
        data: {
            room: activeRoom.value
        }
    }

    send(JSON.stringify(msg));
}

function fetchRooms() {
    const msg = {
        type: EventFetchRooms,
        data: {
        }
    };

    send(JSON.stringify(msg));
}

// Замена Enter на отправку сообщения
editable.addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    }
});

editable.addEventListener('input', updatePlaceholder);

connect();