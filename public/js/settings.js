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