const fileDropZone = document.getElementById("fileDropZone");
const fileInput = document.getElementById("file");
const fileName = document.getElementById("fileName");
const filePreview = document.getElementById("filePreview");
const previewContent = document.getElementById("previewContent");
const uploadForm = document.querySelector(".upload-form");
const uploadProgress = document.getElementById("uploadProgress");
const progressFill = document.getElementById("progressFill");
const progressText = document.getElementById("progressText");
const uploadResult = document.getElementById("uploadResult");
const uploadResultText = document.getElementById("uploadResultText");
const uploadResultLink = document.getElementById("uploadResultLink");
const uploadButton = uploadForm.querySelector(".upload-button");
const uploadTokenInput = document.getElementById("upload_token");
const expiryInput = document.getElementById("expiry_date");
let uploading = false;

function setUploading(state) {
    uploading = state;
    uploadButton.disabled = state;
    fileDropZone.classList.toggle("disabled", state);
    fileInput.disabled = state;
    expiryInput.disabled = state;
    if (uploadTokenInput) uploadTokenInput.disabled = state;
}

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

function showResult(success, message, fileUrl) {
    uploadProgress.style.display = "none";
    setUploading(false);
    uploadResult.style.display = "flex";
    uploadResult.classList.toggle("error", !success);
    uploadResultText.textContent = message;
    uploadResultLink.href = fileUrl || "#";
    uploadResultLink.style.display = success ? "" : "none";
}

function resetUploadState() {
    uploadProgress.style.display = "none";
    uploadResult.style.display = "none";
    progressFill.style.width = "0%";
    progressText.textContent = "0%";
    setUploading(false);
}

uploadForm.addEventListener("submit", (e) => {
    e.preventDefault();

    const formData = new FormData(uploadForm);
    formData.set("plain", "true");
    const xhr = new XMLHttpRequest();

    resetUploadState();
    uploadProgress.style.display = "flex";
    setUploading(true);

    xhr.upload.addEventListener("progress", (e) => {
        if (e.lengthComputable) {
            const percent = Math.round((e.loaded / e.total) * 100);
            progressFill.style.width = percent + "%";
            progressText.textContent = percent + "%";
        }
    });

    xhr.addEventListener("load", () => {
        uploadProgress.style.display = "none";

        if (xhr.status >= 200 && xhr.status < 400) {
            showResult(true, "Upload successful", xhr.responseText.trim());
        } else if (xhr.status === 401) {
            showResult(false, "Upload failed: Unauthorized");
        } else {
            showResult(false, xhr.responseText || "Upload failed");
        }
    });

    xhr.addEventListener("error", () => {
        uploadProgress.style.display = "none";
        showResult(false, "Upload failed");
    });

    xhr.open("POST", "/api/file/upload");
    xhr.send(formData);
});

fileDropZone.addEventListener("click", () => {
    if (uploading) return;
    fileInput.click();
});

fileInput.addEventListener("change", (e) => {
    resetUploadState();
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
    if (uploading) return;

    const files = e.dataTransfer.files;
    if (files.length > 0) {
        resetUploadState();
        fileInput.files = files;
        showFilePreview(files[0]);
    }
});

window.addEventListener('paste', e => {
    if (uploading) return;
    const files = e.clipboardData.files;
    if (files.length > 0) {
        resetUploadState();
        fileInput.files = files;
        showFilePreview(files[0]);
    }
});
