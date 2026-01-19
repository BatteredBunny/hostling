function confirmDeleteUploadTokens(id) {
    if (confirm("Are you sure you want to delete all upload tokens for this user?")) {
        const formData  = new FormData();
        formData.append('id', id);

        fetch('/api/admin/upload_tokens', {
            method: 'DELETE',
            body: formData,
        }).then(response => {
            if (response.ok) {
                alert('All upload tokens for this user have been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete upload tokens for this user.');
            }
        });
    }
}

window.confirmDeleteUploadTokens = confirmDeleteUploadTokens;

function confirmDeleteSessions(id) {
    if (confirm("Are you sure you want to delete all sessions for this user? They will be logged out immediately.")) {
        const formData  = new FormData();
        formData.append('id', id);

        fetch('/api/admin/sessions', {
            method: 'DELETE',
            body: formData,
        }).then(response => {
            if (response.ok) {
                alert('All sessions for this user have been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete sessions for this user.');
            }
        });
    }
}

window.confirmDeleteSessions = confirmDeleteSessions;

function confirmDeleteFiles(id) {
    if (confirm("Are you sure you want to delete all files for this user? This action cannot be undone.")) {
        const formData  = new FormData();
        formData.append('id', id);

        fetch('/api/admin/files', {
            method: 'DELETE',
            body: formData,
        }).then(response => {
            if (response.ok) {
                alert('All files for this user have been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete files for this user.');
            }
        });
    }
}

window.confirmDeleteFiles = confirmDeleteFiles;

function confirmDeleteUser(id) {
    if (confirm("Are you sure you want to delete this user? This action cannot be undone and all their data will be permanently deleted.")) {
        const formData  = new FormData();
        formData.append('id', id);

        fetch('/api/admin/user', {
            method: 'DELETE',
            body: formData,
        }).then(response => {
            if (response.ok) {
                alert('The user has been deleted.');
                window.location.reload();
            } else {
                alert('Failed to delete user.');
            }
        });
    }
}

window.confirmDeleteUser = confirmDeleteUser;

function giveInvite(id) {
    const formData  = new FormData();
    formData.append('id', id);

    fetch('/api/admin/give_invite_code', {
        method: 'POST',
        body: formData,
    }).then(response => {
        if (response.ok) {
            alert('An invite code has been given to the user.');
            window.location.reload();
        } else {
            alert('Failed to give invite code to user.');
        }
    });
}

window.giveInvite = giveInvite;