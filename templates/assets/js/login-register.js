/*
 *
 * login-register modal
 * 
 */
function showRegisterForm(){
    $('.loginBox').hide();
    $('.registerBox').fadeIn('fast');
    $('.login-footer').hide();
    $('.register-footer').fadeIn('fast');
    $('.modal-title').html('Register with');
    $('.error').removeClass('alert alert-danger').html('');
}
function showLoginForm(){
    $('.registerBox').hide();
    $('.loginBox').fadeIn('fast');
    $('.register-footer').hide();
    $('.login-footer').fadeIn('fast');
    $('.modal-title').html('Login with');
    $('.error').removeClass('alert alert-danger').html('');
}

function openLoginModal(){
    showLoginForm();
    setTimeout(function(){
        $('#loginModal').modal('show');    
    }, 230);
    
}
function openRegisterModal(){
    showRegisterForm();
    setTimeout(function(){
        $('#loginModal').modal('show');    
    }, 230);
    
}

function loginAjax() {
    var email = $(".loginBox #email").val();
    var password = $(".loginBox #password").val();
    if (!email || !password) {
        $(".loginBox .error").text("Please enter your email and password.");
        return;
    }
    $.ajax({
        url: '/api/login',
        type: 'POST',
        contentType: 'application/json',
        data: JSON.stringify({email: email, password: password}),
        success: function(data) {
            $(".loginBox .error").text("");
            $('#loginModal').modal('hide');
            location.reload();
        },
        error: function(xhr) {
            $(".loginBox .error").text(xhr.responseText);
        }
    });
}

function shakeModal(){
    $('#loginModal .modal-dialog').addClass('shake');
             $('.error').addClass('alert alert-danger').html("Invalid email/password combination");
             $('input[type="password"]').val('');
             setTimeout( function(){ 
                $('#loginModal .modal-dialog').removeClass('shake'); 
    }, 1000 ); 
}

function registerAjax() {
    console.log("registerAjax called");
    
    // Get values from the hidden form
    var email = $("#reg_email").val();
    var password = $("#reg_password").val();
    var password_confirmation = $("#password_confirmation").val();
    
    console.log("Email:", email);
    console.log("Password length:", password.length);
    
    if (email == "" || password == "") {
        console.log("Missing fields");
        $(".registerBox .error").fadeIn(400).html("Please fill all fields");
        return false;
    }
    
    if (password != password_confirmation) {
        console.log("Passwords don't match");
        $(".registerBox .error").fadeIn(400).html("Passwords don't match");
        return false;
    }
    
    console.log("Making registration request");
    $.ajax({
        url: "/api/register",
        type: "POST",
        contentType: "application/json",
        data: JSON.stringify({
            email: email,
            password: password
        }),
        success: function(response) {
            console.log("Registration successful:", response);
            $(".registerBox").hide();
            $(".otpBox").fadeIn("fast");
            $("#otp_email").val(email);
        },
        error: function(xhr, status, error) {
            console.log("Registration failed:", error);
            console.log("Response:", xhr.responseText);
            var errMsg = xhr.responseJSON?.error || "Registration failed";
            if (xhr.status === 409 && errMsg.includes("already exists")) {
                $(".registerBox .error").fadeIn(400).html('<span style="color:#d32f2f;font-style:italic;">' + errMsg + '</span>');
            } else {
                $(".registerBox .error").fadeIn(400).html(errMsg);
            }
        }
    });
}

function verifyOtpAjax() {
    console.log("verifyOtpAjax called");
    
    var email = $("#otp_email").val();
    var otp = $("#otp_code").val();
    
    if (otp == "") {
        console.log("Empty OTP");
        $(".otpBox .error").fadeIn(400).html("Please enter OTP");
        return false;
    }
    
    console.log("Making OTP verification request");
    $.ajax({
        url: "/api/verify-otp",
        type: "POST",
        contentType: "application/json",
        data: JSON.stringify({
            email: email,
            otp: otp
        }),
        success: function(response) {
            console.log("OTP verification successful:", response);
            window.location.href = "/dashboard";
        },
        error: function(xhr, status, error) {
            console.log("OTP verification failed:", error);
            console.log("Response:", xhr.responseText);
            $(".otpBox .error").fadeIn(400).html(xhr.responseJSON?.error || "OTP verification failed");
        }
    });
}

$(document).ready(function() {
    // Modal openers
    $(document).on('click', '.login-btn', function(e) {
        e.preventDefault();
        openLoginModal();
    });
    $(document).on('click', '.signup-btn', function(e) {
        e.preventDefault();
        openRegisterModal();
    });
    // Modal footer links
    $(document).on('click', '.show-register', function(e) {
        e.preventDefault();
        showRegisterForm();
    });
    $(document).on('click', '.show-login', function(e) {
        e.preventDefault();
        showLoginForm();
    });
    // Login
    $(document).on('click', '.btn-login', function(e) {
        e.preventDefault();
        loginAjax();
    });
    // Register
    $(document).on('click', '.btn-register', function(e) {
        e.preventDefault();
        registerAjax();
    });
    // OTP
    $(document).on('click', '.btn-verify-otp', function(e) {
        e.preventDefault();
        verifyOtpAjax();
    });
    // New button (reset form)
    $(document).on('click', '.btn-new', function(e) {
        e.preventDefault();
        resetForm();
    });
    // Copy shortened URL (main form)
    $(document).on('click', '.btn-copy', function(e) {
        e.preventDefault();
        var url = $(this).data('url') || $('#shortenedURL').val();
        if (url) {
            navigator.clipboard.writeText(url);
            if (typeof showCopiedMessage === 'function') showCopiedMessage();
        }
    });
    // QR code (main form)
    $(document).on('click', '.btn-qrcode', function(e) {
        e.preventDefault();
        var url = $(this).data('url') || $('#shortenedURL').val();
        if (url) {
            var qrcode = $('#qrcode');
            qrcode.html('<img src="/api/qrcode?data=' + encodeURIComponent(url) + '&size=150" alt="QR Code" />');
        }
    });
    // Share button
    $(document).on('click', '.btn-share', function(e) {
        e.preventDefault();
        var url = $(this).data('url');
        if (navigator.share && url) {
            navigator.share({ url: url });
        } else if (url) {
            navigator.clipboard.writeText(url);
            if (typeof showCopiedMessage === 'function') showCopiedMessage();
        }
    });
    // Show QR code in history
    $(document).on('click', '.btn-qrcode', function(e) {
        var url = $(this).data('url');
        if (url) {
            var span = $(this).closest('div').find('.qr-code-span');
            span.html('<img src="/api/qrcode?data=' + encodeURIComponent(url) + '&size=100" alt="QR Code" style="vertical-align:middle; margin-left:10px;" />');
        }
    });
});