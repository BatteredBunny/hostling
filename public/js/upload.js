const fileDropZone = document.getElementById("fileDropZone");
const fileInput = document.getElementById("file");
const fileName = document.getElementById("fileName");
const filePreview = document.getElementById("filePreview");
const previewContent = document.getElementById("previewContent");

function showFilePreview(file) {
    fileName.textContent = file.name;
    previewContent.innerHTML = "";

    const fileType = file.type.toLowerCase();
    const fileURL = URL.createObjectURL(file);

    if (fileType.startsWith("image/")) {
        const img = document.createElement("img");
        img.src = fileURL;
        img.alt = file.name;
        previewContent.appendChild(img);
    } else if (fileType.startsWith("video/")) {
        const video = document.createElement("video");
        video.src = fileURL;
        previewContent.appendChild(video);
    } else if (fileType.startsWith("audio/")) {
        const audio = document.createElement("audio");
        audio.controls = true;
        audio.src = fileURL;
        previewContent.appendChild(audio);
    }

    filePreview.style.display = "flex";
    fileDropZone.classList.add("file-selected");
}

fileDropZone.addEventListener("click", () => {
    fileInput.click();
});

fileInput.addEventListener("change", (e) => {
    const file = e.target.files[0];
    if (file) {
        showFilePreview(file);
    }
});

fileDropZone.addEventListener("dragover", (e) => {
    e.preventDefault();
    fileDropZone.classList.add("drag-over");
});

fileDropZone.addEventListener("dragleave", (e) => {
    e.preventDefault();
    fileDropZone.classList.remove("drag-over");
});

fileDropZone.addEventListener("drop", (e) => {
    e.preventDefault();
    fileDropZone.classList.remove("drag-over");

    const files = e.dataTransfer.files;
    if (files.length > 0) {
        fileInput.files = files;
        showFilePreview(files[0]);
    }
});

window.addEventListener('paste', e => {
    const files = e.clipboardData.files;
    if (files.length > 0) {
        fileInput.files = files;
        showFilePreview(files[0]);
    }
});