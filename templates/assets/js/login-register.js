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

function loginAjax(){
    /*   Remove this comments when moving to server
    $.post( "/login", function( data ) {
            if(data == 1){
                window.location.replace("/home");            
            } else {
                 shakeModal(); 
            }
        });
    */

/*   Simulate error message from the server   */
     shakeModal();
}

function shakeModal(){
    $('#loginModal .modal-dialog').addClass('shake');
             $('.error').addClass('alert alert-danger').html("Invalid email/password combination");
             $('input[type="password"]').val('');
             setTimeout( function(){ 
                $('#loginModal .modal-dialog').removeClass('shake'); 
    }, 1000 ); 
}

   