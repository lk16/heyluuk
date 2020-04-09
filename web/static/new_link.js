
function load_new_challenge() {
    $.ajax({
        type: "GET",
        url: '/api/challenge',
        success: function (result) {
            $("#form-challenge-question").html(result.question);
            $("#form-challenge-id").attr("value", result.id);
        }
    });
}

function nodes_api_to_array(data) {
    data = data.sort(function(a,b){
        if(a.path_segment < b.path_segment) {
            return -1;
        }
        return 1;
    });

    var children = [];
    $(data).each(function(_index, element){

        var text = element.path_segment;

        if(element['url'] !== '') {
            text = '<a href="' + element['url'] + '">' + text + '</a>';
        }

        children.push({
            text: text,
            lazyLoad: true,
            id: element.id,
            selectable: false
        });
    });

    return children
}

function lazy_load_tree_nodes(node, dataHandler) {
    $.ajax({
        type: 'GET',
        url: '/api/node/' + node.id + '/children',
        success: function(data, _status, _xhr) {
            children = nodes_api_to_array(data);
            dataHandler(children);
        }
    });
}

function load_link_tree() {
    $.ajax({
        url: '/api/node/root',
        success: function(data, _status, _xhr) {

            var treeData = [
                {
                    text: 'heylu.uk',
                    nodes: nodes_api_to_array(data),
                    selectable: false,
                }
            ];

            $('#myTree').treeview({
                data: treeData,
                lazyLoad: lazy_load_tree_nodes,
                showBorder: false,
                expandIcon: "fas fa-plus",
                collapseIcon: "fas fa-minus",
                loadingIcon: "fas fa-ellipsis-v",
                emptyIcon: "fas fa-link"
            });
        }
    });
}

$(document).ready(function () {

    load_new_challenge();
    load_link_tree();

    $("#new-link-form").submit(function (_e) {

        var array = $("#new-link-form").serializeArray();
        var body = {};
        $(array).each(function (_index, obj) {
            body[obj.name] = obj.value;
        });

        $.ajax({
            type: "POST",
            contentType: "application/json; charset=utf-8",
            url: '/api/link',
            data: JSON.stringify(body),
            success: function (result) {
                var message = "Your link <a target='_blank' href='" + result['shortcut'] + "'>" + window.location.host + result['shortcut'] + "</a> has been created.";
                $("#form-alert").html(message).removeClass("alert-danger").addClass("alert-success").show();
                $("#new-link-form").find("input").val("");
                load_new_challenge();
                load_link_tree();
            },
            error: function (xhr, _resp, _text) {
                var message = "Error: " + xhr.responseJSON["error"];
                $("#form-alert").html(message).removeClass("alert-success").addClass("alert-danger").show();
                $("#form-challenge-answer").val("");
                load_new_challenge();
            }
        });

        return false;
    });
});