jQuery.loadScript = function (url, callback) {
    jQuery.ajax({
        url: url,
        dataType: 'script',
        success: callback,
        async: true
    });
};

var mime = 'text/plain';

var languageMimeArray = {
    "text/generic": "generic",
    "text/javascript": "javascript",
    "text/x-csrc": "clike",
    "text/x-cmake": "cmake",
    "text/css": "css",
    "text/x-d": "d",
    "text/x-diff": "diff",
    "text/x-dockerfile": "dockerfile",
    "text/x-erlang": "erlang",
    "text/x-go": "go",
    "text/x-haskell": "haskell",
    "text/html": "xml",
    "text/x-java": "clike",
    "text/x-kotlin": "clike",
    "jinja2": "jinja2",
    "text/x-lua": "lua",
    "text/x-markdown": "markdown",
    "text/x-perl": "perl",
    "text/x-php": "php",
    "text/x-python": "python",
    "text/x-rpm-changes": "rpm",
    "text/x-rst": "rst",
    "text/x-ruby": "ruby",
    "text/x-rust": "rust",
    "text/x-sh": "shell",
    "text/x-sql": "sql",
    "text/x-swift": "swift",
    "text/x-yaml": "yaml",
    "application/xml": "xml",
    "text/plain": "text"
};

var editor = CodeMirror.fromTextArea(document.getElementById("paste"), {
    lineNumbers: true,
    viewportMargin: Infinity,
    matchBrackets: true,
    lineWrapping: true,
    scrollbarStyle: "simple",
    theme: "one-dark",
    mode: "text/plain"
});


function setEditorSyntax(mime) {
    if (mime === "text/javascript" || mime === "text/generic") {
        editor.setOption('mode', "text/javascript");
    } else {
        $.loadScript('/assets/js/mode/'+languageMimeArray[mime]+'.js', function() {
            editor.setOption('mode', mime);
        });
    }
}


$(document).ready(function(){
    var expiredValue = 0;
    var lastCall = -1;
    var darkTheme = true;
    checkUserSetColorScheme();

    $('.tooltipped').tooltip({delay: 20});

    function checkUserSetColorScheme() {
        if (Cookies.enabled) {
            if(Cookies.get('darktheme') === "no") {
                darkTheme = false;
                setColorScheme();
            }
        }
    }

    $('.language-item').click(function(e) {
        $('.language-item.active').removeClass('active');
        $(this).addClass('active');
        $('#selectedLanguage').html('<i class="material-icons left">code</i>'+$(e.target).text());
        mime = $(this).data('value');

        switch(mime){
            case 'text/plain':
                editor.setOption('mode', null);
                break;
            default:
                setEditorSyntax(mime);
                break;
        }
    });

    $('.expire-item').click(function(e) {
        $('.expire-item.active').removeClass('active');
        $(this).addClass('active');
        $('#selectedExpiration').text($(e.target).text());
        expiredValue = $(this).data('value');
    });

    $('#upload').click(function(e){
        e.preventDefault();
        if (editor.getValue()) {
            if (lastCall + 1000 < Date.now()) {
                lastCall = Date.now();
                callAjax()
            }
        } else {
            var $toastContent = $('<span>Error: Your paste is empty</span>');
            Materialize.toast($toastContent, 2000);
        }
    });


    $('#changeTheme').click(function(e) {
        darkTheme = !darkTheme;
        value = darkTheme ? "yes" : "no";
        Cookies.set('darktheme', value);
        setColorScheme();
    });

    function setColorScheme() {
        if(darkTheme) {
            $('body').removeClass('light-theme');
            editor.setOption("theme","one-dark");
        } else {
            $('body').addClass('light-theme');
            editor.setOption("theme","duotone-light");
        }
    }

    function callAjax() {
        $.ajax({
            url: '/',
            type: 'POST',
            data: {
                p: editor.getValue(),
                title: $('#title').val(),
                expire: expiredValue,
                mime: mime,
                raw: "0"
            }
        }).done(function(data){
            window.location = data;
        }).fail(function (xhr){
            // var json = $.parseJSON(xhr.responseText);
            var $toastContent = $('<span>Error: '+xhr.responseText+'</span>');
            Materialize.toast($toastContent, 5000);
        });
    }
});