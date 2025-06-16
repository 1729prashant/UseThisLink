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
    var email = $(".registerBox #email").val();
    var password = $(".registerBox #password").val();
    var password_confirmation = $(".registerBox #password_confirmation").val();
    if (!email || !password || password !== password_confirmation) {
        $(".registerBox .error").text("Please enter valid email and matching passwords.");
        return;
    }
    $.ajax({
        url: '/api/register',
        type: 'POST',
        contentType: 'application/json',
        data: JSON.stringify({email: email, password: password}),
        success: function(data) {
            $(".registerBox").hide();
            $(".otpBox").show();
            $("#otp_email").val(email);
            $(".registerBox .error").text("");
        },
        error: function(xhr) {
            $(".registerBox .error").text(xhr.responseText);
        }
    });
}
function verifyOtpAjax() {
    var email = $("#otp_email").val();
    var otp = $("#otp_code").val();
    if (!otp) {
        $(".otpBox .error").text("Please enter the OTP.");
        return;
    }
    $.ajax({
        url: '/api/verify-otp',
        type: 'POST',
        contentType: 'application/json',
        data: JSON.stringify({email: email, otp: otp}),
        success: function(data) {
            $(".otpBox").hide();
            alert("Registration successful! You can now log in.");
            showLoginForm();
        },
        error: function(xhr) {
            $(".otpBox .error").text(xhr.responseText);
        }
    });
}
$(function() {
    $(".btn-register").off('click').on('click', registerAjax);
});