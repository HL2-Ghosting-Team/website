var ghostingServices = angular.module('ghostingServices', []);

ghostingServices.factory('GhostingAPI', function() {
	return gapi.client.ghosting;
});

var CLIENT_ID = '434185101696.apps.googleusercontent.com';

var SCOPES = 'https://www.googleapis.com/auth/userinfo.email';

var RESPONSE_TYPE = 'token id_token';

ghostingServices.factory('Auth', ['$rootScope', function($rootScope) {
	var library, authenticateCallback;
	
	authenticateCallback = function() {
		library.checkAuthenticated();
	};

	var init = function() {
		$rootScope.auth = library;
		$rootScope.auth.loaded = false;
		$rootScope.auth.user = {
			'signed_in': false
		};
		library.authenticate(true);
	};

	library = {
		'authenticate': function(immediate) {
			gapi.auth.authorize({
				client_id: CLIENT_ID,
				scope: SCOPES,
				immediate: immediate,
				response_type: RESPONSE_TYPE
			}, authenticateCallback)
		},
		'checkAuthenticated': function() {
			gapi.client.ghosting.users.get({'user': 'current'}).execute(function(resp) {
				if (!resp.code) {
					var token = gapi.auth.getToken();
					token.access_token = token.id_token;
					gapi.auth.setToken(token);

					$rootScope.auth.user = {
						'signed_in': true,

						'id': resp.id,
						'nickname': resp.nickname,
						'avatar': resp.avatar,
						'admin': resp.admin
					};
					$rootScope.auth.loaded = true;
					$rootScope.$apply();
				} else {
					$rootScope.auth.user = {
						'signed_in': false
					};
					$rootScope.auth.loaded = false;
					$rootScope.$apply();
				}
			});
		},
		'unauthenticate': function() {
			gapi.auth.setToken(null);
			$rootScope.auth.user = {
				'signed_in': false
			}
			$rootScope.auth.loaded = true;
			$rootScope.$apply();
		}
	};

	if($rootScope.auth == null) {
		init();
	}

	return library;
}]);