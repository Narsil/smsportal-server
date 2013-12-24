(function(){
    var make_dom_id = function(destinator){
        return 'destinator-' + destinator.replace('+', '').replace(' ', '');
    };
    var createTabs = function(destinators, last_destinator){
        var list = $('<ul></ul>')
            .addClass('nav nav-tabs');
        var tab_content = $('<div></div>')
            .addClass('tab-content');
        var tab_pane = $('<div></div>')
            .addClass('tab-pane');
        var item;
        var link;
        $.each(destinators, function(destinator, messages){
            link = $('<a></a>')
                .attr('href', '#' + make_dom_id(destinator))
                .attr('data-toggle', 'tab')
                .text(destinator);
            item = $('<li></li>').append(link);
    
            var tab_pane_ = tab_pane.clone();
            tab_pane_.attr('id', make_dom_id(destinator));
            if (destinator == last_destinator){
                tab_pane_.addClass('active')
            }

            var form = $('<form></form>')
                .css({'float':'right'})
                .attr('action', '.')
                .attr('method', 'post')
                .append(
                    $('<input></input>')
                        .attr('type', 'hidden')
                        .attr('value', destinator)
                        .attr('name', 'To')
                    ,$('<input></input>')
                        .attr('type', 'text')
                        .attr('placeholder', 'Message...')
                        .attr('name', 'Message')
                        .css({'height': '30px'}));

            var pane_list = $('<ul></ul>').css({
                'list-style': 'none',
                'margin-right': '25px'});
            pane_list.append(
                    form,
                    $('<li></li>').css({'clear': 'both'}));
            $.each(messages, function(index, message){
                var span = $('<span></span>')
                        .css({'display': 'inline-block'})
                        .addClass('well')
                        .text(message.Message);
                if (!message.Incoming){
                    span.css({'float': 'right'});
                    if (message.Sent){
                        span.addClass('alert alert-success');
                    }else{
                        span.addClass('alert alert-info');
                    }
                }
                pane_list.append(
                    $('<li></li>')
                        .append(span), 
                    $('<li></li>').css({'clear': 'both'}));
            });
            tab_pane_.append(pane_list);

            tab_content.append(tab_pane_)
            list.append(item)
        });
        $('body').append(list, tab_content);
    };
    var parseHistory = function(data){
        var destinators = {}
        var destinator;
        var message;
        for (var i = 0; i < data.length; i++){
            message = data[i];
            console.log(message.Incoming);
            if (message.Incoming){
                destinator = message.From;
            }else{
                destinator = message.To;
            }
            if (!destinators.hasOwnProperty(destinator)){
                destinators[destinator] = [];
            }
            destinators[destinator].push(message);
        }
        var last_destinator;
        if (data[0].Incoming){
            last_destinator = data[0].From;
        }else{
            last_destinator = data[0].To;
        }
        createTabs(destinators, last_destinator);

    };
    $.getJSON('/history/', parseHistory);
})($);
