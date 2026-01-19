const deleteButtons = document.querySelectorAll('.delete-button');

for (let button of deleteButtons) {
    if (!button.dataset.confirm) {
        continue;
    }

    button.addEventListener('click', function(e) {
        const confirmMessage = e.target.dataset.confirm;

        if (!confirm(confirmMessage)) {
            e.preventDefault();
        }
    });
}

function confirmDeleteAllFiles() {
    const confirmMessage = "Are you sure you want to delete ALL your files? This action cannot be undone.";

    if (confirm(confirmMessage)) {
        fetch('/api/account/files', {
            method: 'DELETE',
        }).then(response => {
            if (response.ok) {
                alert('All files have been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete files.');
            }
        });
    }
}

window.confirmDeleteAllFiles = confirmDeleteAllFiles;

function confirmDeleteAccount() {
    const confirmMessage = "Are you sure you want to delete your account? This action cannot be undone and all your data will be permanently deleted.";

    if (confirm(confirmMessage)) {
        fetch('/api/account', {
            method: 'DELETE',
        }).then(response => {
            if (response.ok) {
                alert('Your account has been deleted.');
                window.location.href = '/';
            } else {
                alert('Failed to delete account.');
            }
        });
    }
}

window.confirmDeleteAccount = confirmDeleteAccount;

function confirmDeleteInvite(code) {
    const confirmMessage = "Are you sure you want to delete this invite code?";

    if (confirm(confirmMessage)) {
        const formData  = new FormData();
        formData.append('invite_code', code);

        fetch('/api/account/invite_code', {
            method: 'DELETE',
            body: formData,
        }).then(response => {
            if (response.ok) {
                alert('Invite code has been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete invite code.');
            }
        });
    }
}

window.confirmDeleteInvite = confirmDeleteInvite;

function confirmDeleteUploadToken(token) {
    const confirmMessage = "Are you sure you want to delete this upload token?";

    if (confirm(confirmMessage)) {
        const formData  = new FormData();
        formData.append('upload_token', token);

        fetch('/api/account/upload_token', {
            method: 'DELETE',
            body: formData,
        }).then(response => {
            if (response.ok) {
                alert('Upload token has been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete upload token.');
            }
        });
    }
}

window.confirmDeleteUploadToken = confirmDeleteUploadToken;